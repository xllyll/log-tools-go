# 日志查询性能优化指南

## 问题描述

当前 `/api/logs` 接口在查询8个文件、按模块过滤（26619条匹配数据）时，响应时间超过2秒。

## 性能瓶颈分析

### 1. 索引缺失
- **原有索引**: 只有单列索引 `file_id`, `log_time`, `level`, `source`
- **查询条件**: `file_id IN (...) AND module = ? ORDER BY log_time`
- **问题**: 缺少复合索引，无法高效支持多条件查询和排序

### 2. SQL查询效率低
- WHERE条件顺序不合理（低选择性条件在前）
- LIKE '%keyword%' 无法使用索引
- 多次执行相似查询（GetLogEntries + GetLogStats）

### 3. 数据库配置
- SQLite默认配置未针对大数据量优化
- 缺少统计信息（ANALYZE）

## 优化方案

### ✅ 已实施的优化

#### 1. 添加复合索引（最重要）

```sql
-- 模块查询优化索引
CREATE INDEX idx_log_entries_module ON log_entries(module);

-- 文件+模块+时间复合索引（核心优化）
CREATE INDEX idx_log_entries_file_module_time ON log_entries(file_id, module, log_time);

-- 文件+时间复合索引
CREATE INDEX idx_log_entries_file_time ON log_entries(file_id, log_time);

-- 文件+级别复合索引
CREATE INDEX idx_log_entries_file_level ON log_entries(file_id, level);
```

**原理**: 
- 复合索引 `(file_id, module, log_time)` 可以完美支持你的查询：
  ```sql
  WHERE file_id IN (...) AND module = 'DeviceService' 
  ORDER BY log_time
  ```
- 索引覆盖所有查询条件和排序字段，避免回表查询

#### 2. 优化WHERE条件顺序

调整过滤条件的添加顺序，将高选择性条件放在前面：
```go
// 优先：module (等值查询，选择性高)
if filter.Module != "" {
    query += " AND module = ?"
}

// 其次：levels (IN查询)
if len(filter.Levels) > 0 {
    query += " AND level IN (...)"
}

// 再次：时间范围 (范围查询)
if filter.StartTime != nil {
    query += " AND log_time >= ?"
}

// 最后：LIKE查询 (效率最低)
if len(filter.Keywords) > 0 {
    query += " AND content LIKE '%keyword%'"
}
```

#### 3. 代码重构优化

- 提取 `buildWhereClause` 辅助函数，避免重复代码
- 移除生产环境的SQL日志打印（减少I/O开销）
- 统一过滤条件构建逻辑

### 📋 待执行的优化步骤

#### 步骤1: 为现有数据库添加索引

运行迁移脚本：
```bash
go run migrate_indexes.go
```

或手动执行SQL：
```bash
sqlite3 logs_v1.db <<EOF
CREATE INDEX IF NOT EXISTS idx_log_entries_module ON log_entries(module);
CREATE INDEX IF NOT EXISTS idx_log_entries_file_module_time ON log_entries(file_id, module, log_time);
CREATE INDEX IF NOT EXISTS idx_log_entries_file_time ON log_entries(file_id, log_time);
CREATE INDEX IF NOT EXISTS idx_log_entries_file_level ON log_entries(file_id, level);
ANALYZE;
VACUUM;
EOF
```

#### 步骤2: 重启应用

新索引会在应用启动时自动创建（database.go中的initTables已更新）。

#### 步骤3: 验证性能

测试相同查询的响应时间：
```bash
curl -X POST http://127.0.0.1:4080/api/logs \
  -H "Content-Type: application/json" \
  -d '{
    "file_ids": "bfacce6d,6b757bb2,26c4dfdb,e0ae40ce,aac0c932,a4882d15,b3f61e1e,2318833a",
    "limit": 100,
    "offset": 0,
    "module": "DeviceService"
  }'
```

预期结果：**响应时间从 2s+ 降低到 200ms 以内**（提升10倍以上）

## 进一步优化建议

### 短期优化（1-2天）

#### 1. SQLite配置优化

在 `internal/model/database.go` 的 `NewDatabase` 函数中添加：

```go
func NewDatabase(cfg *config.Config) (*Database, error) {
    // ... 现有代码 ...
    
    // SQLite性能优化配置
    optimizations := []string{
        `PRAGMA journal_mode = WAL`,           // WAL模式提高并发性能
        `PRAGMA synchronous = NORMAL`,         // 平衡安全性和性能
        `PRAGMA cache_size = -64000`,          // 64MB缓存
        `PRAGMA temp_store = MEMORY`,          // 临时表使用内存
        `PRAGMA mmap_size = 268435456`,        // 256MB内存映射
    }
    
    for _, opt := range optimizations {
        if _, err := db.Exec(opt); err != nil {
            log.Printf("SQLite配置失败: %v", err)
        }
    }
    
    // ... 现有代码 ...
}
```

#### 2. 分页查询优化

对于大偏移量分页（如 offset=10000），使用游标分页：

```go
// 替代 OFFSET 的方式
query += " AND (log_time > ? OR (log_time = ? AND line_number > ?))"
args = append(args, lastLogTime, lastLogTime, lastLineNumber)
```

#### 3. 连接池配置

```go
db.SetMaxOpenConns(25)
db.SetMaxIdleConns(10)
db.SetConnMaxLifetime(5 * time.Minute)
```

### 中期优化（1周）

#### 1. 缓存热门查询结果

使用内存缓存存储常用查询的统计信息：

```go
import "sync"

type StatsCache struct {
    mu    sync.RWMutex
    cache map[string]struct {
        stats LogStats
        expire time.Time
    }
}
```

#### 2. 异步统计计算

将 `GetLogStats` 改为异步执行，先返回日志列表，统计信息稍后推送。

#### 3. 全文搜索优化

如果关键词搜索频繁，考虑：
- 使用 SQLite FTS5 扩展
- 或集成 Elasticsearch

### 长期优化（1月+）

#### 1. 数据分片

按日期或项目拆分数据库文件：
```
logs_2026_05.db
logs_2026_04.db
```

#### 2. 读写分离

- 主数据库：写入
- 只读副本：查询（使用 SQLite WAL 模式的快照）

#### 3. 迁移到 PostgreSQL

当数据量超过百万级时，考虑迁移到更强大的数据库。

## 性能监控

### 添加查询耗时日志

在 `internal/handler/log.go` 中：

```go
func (h *LogHandler) GetLogs(c *gin.Context) {
    startTime := time.Now()
    
    // ... 现有逻辑 ...
    
    elapsed := time.Since(startTime)
    log.Printf("查询耗时: %v, 返回 %d 条记录", elapsed, len(entries))
    
    c.JSON(http.StatusOK, model.LogResponse{
        Success: true,
        Data:    entries,
        Stats:   stats,
    })
}
```

### 关键指标

- **P50响应时间**: < 100ms
- **P95响应时间**: < 500ms  
- **P99响应时间**: < 1000ms

## 总结

| 优化项 | 预期提升 | 实施难度 | 优先级 |
|--------|---------|---------|--------|
| 添加复合索引 | 10-50x | 低 | ⭐⭐⭐⭐⭐ |
| 优化WHERE顺序 | 1.5-2x | 低 | ⭐⭐⭐⭐ |
| SQLite配置优化 | 2-3x | 低 | ⭐⭐⭐⭐ |
| 移除SQL日志 | 1.2x | 低 | ⭐⭐⭐ |
| 缓存统计信息 | 5-10x (缓存命中) | 中 | ⭐⭐⭐ |
| 游标分页 | 2-5x (大offset) | 中 | ⭐⭐ |

**立即执行前两项即可获得显著性能提升！**
