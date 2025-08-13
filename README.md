# 日志分析工具

一个基于Go语言开发的本地日志分析系统，提供Web界面进行日志文件上传、解析、分析和查询。

## ✨ 功能特性

- 🌐 **Web界面**: 基于Bootstrap的现代化响应式界面
- 📁 **文件上传**: 支持上传 `.txt`, `.log`, `.gz`, `.zip` 格式的日志文件
- 🔄 **自动解压**: 自动解压缩 `.gz` 和 `.zip` 文件
- 🎨 **颜色编码**: 根据日志级别显示不同颜色（ERROR红色、WARN黄色、INFO蓝色等）
- 🔍 **智能过滤**: 支持按日志级别、关键词、时间范围进行过滤
- 📊 **统计分析**: 实时显示日志统计信息（总数、各级别数量等）
- 🗄️ **SQLite数据库**: 日志数据存储在本地SQLite数据库中，支持复杂查询
- 📱 **响应式设计**: 适配桌面和移动设备
- 🔄 **折叠显示**: 日志记录默认显示一行，点击可展开查看完整内容
- 📄 **分页浏览**: 支持大量日志的分页显示
- 🔍 **全文搜索**: 支持在日志内容中进行关键词搜索

## 🚀 快速开始

### 环境要求

- Go 1.24 或更高版本
- Windows/Linux/macOS

### 安装和运行

1. **克隆项目**
   ```bash
   git clone https://github.com/xllyll/log-tools-go.git
   cd log-tools-go
   ```

2. **安装依赖**
   ```bash
   go mod tidy
   ```

3. **运行程序**
   ```bash
   go run main.go
   ```

4. **访问界面**
   打开浏览器访问 `http://127.0.0.1:4080`

### 打包成可执行文件

```bash
# Windows
go build -ldflags="-s -w" -o log-analyzer.exe

# Linux/macOS
go build -ldflags="-s -w" -o log-analyzer
```

## 📁 项目结构

```
log-tools-go/
├── main.go                 # 主程序入口
├── go.mod                  # Go模块文件
├── config/
│   ├── conf.yaml          # 配置文件
│   └── config.json        # 配置json
├── internal/
│   ├── config/
│   │   └── config.go      # 配置结构定义
│   ├── model/
│   │   ├── log.go         # 数据模型定义
│   │   └── database.go    # 数据库操作
│   ├── service/
│   │   ├── parser.go      # 日志解析服务
│   │   └── storage.go     # 存储服务
│   └── handler/
│       ├── upload.go      # 文件上传处理器
│       └── log.go         # 日志查询处理器
├── web/
│   └── router.go         # 路由定义
├── pkg/
│   ├── xtime/
│   │   └── time.go         # 时间处理工具
│   └── xmatch/
│       └── match.go        # 正则匹配工具
├── web/
│   ├── templates/
│   │   └── index.html     # 主页面模板
│   └── static/            # 静态资源文件
├── uploads/               # 上传文件存储目录
├── logs/                  # 解析后日志存储目录
├── logs.db               # SQLite数据库文件
├── test.log              # 测试日志文件
├── build.bat             # Windows构建脚本
└── README.md             # 项目说明文档
```

## ⚙️ 配置说明

### 配置文件 (config/config.yaml)

```yaml
server:
  port: 4080
  host: "127.0.0.1"

storage:
  upload_dir: "./uploads"
  log_dir: "./logs"
  max_file_size: 104857600  # 100MB
  database_path: "./logs.db"  # SQLite数据库文件路径

log_levels:
  ERROR: "#dc3545"    # 红色
  WARN: "#ffc107"     # 黄色
  INFO: "#17a2b8"     # 蓝色
  DEBUG: "#6c757d"    # 灰色
  FATAL: "#721c24"    # 深红色
  TRACE: "#6f42c1"    # 紫色

filters:
  default_levels: ["ERROR", "WARN", "INFO"]
  exclude_patterns: []
  include_patterns: []

log_patterns:
  - name: "standard"
    pattern: (\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2}(?:\.\d+)?)\s+(\w+)\s+(.+)
    time_format: "2006-01-02 15:04:05"
  - name: "json"
    pattern: (\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d+)?Z?)\s+(\w+)\s+(.+)
    time_format: "2006-01-02T15:04:05Z07:00"
```

## 🗄️ 数据库功能

系统使用SQLite数据库存储日志数据，提供以下功能：

- **持久化存储**: 日志数据永久保存在本地数据库中
- **快速查询**: 支持复杂的SQL查询和过滤
- **统计分析**: 实时计算日志统计信息
- **全文搜索**: 在日志内容中进行关键词搜索

### 数据库测试

运行以下命令测试数据库功能：

```bash
go run test_db.go
```

## 🎨 界面特性

### 日志显示优化

- **折叠显示**: 日志记录默认显示一行，节省界面空间
- **点击展开**: 点击日志条目可展开查看完整内容
- **渐变效果**: 折叠状态下显示渐变遮罩，提示有更多内容
- **动画过渡**: 展开/折叠过程有平滑的动画效果
- **视觉指示**: 展开状态有旋转的箭头指示器

### 响应式设计

- **桌面优化**: 大屏幕下显示完整的侧边栏和主内容区
- **移动适配**: 小屏幕下自动调整布局
- **触摸友好**: 支持触摸设备的点击操作

## 🔧 使用说明

### 上传日志文件

1. 点击上传区域或拖拽文件到指定区域
2. 支持的文件格式：`.txt`, `.log`, `.gz`, `.zip`
3. 系统会自动解压缩并解析日志内容

### 查看和分析日志

1. 在左侧文件列表中选择要查看的日志文件
2. 使用过滤设置筛选特定条件的日志
3. 点击日志条目展开查看完整内容
4. 使用搜索功能查找特定关键词

### 过滤和搜索

- **日志级别**: 选择要显示的日志级别
- **关键词**: 在日志内容中搜索特定关键词
- **时间范围**: 按时间范围过滤日志
- **实时更新**: 过滤条件实时应用到显示结果

## 🛠️ 故障排除

### 常见问题

1. **端口被占用**
   - 修改 `config.yaml` 中的端口号
   - 或关闭占用端口的程序

2. **文件上传失败**
   - 检查文件大小是否超过限制
   - 确认文件格式是否支持
   - 查看控制台错误信息

3. **数据库连接失败**
   - 确保有写入权限
   - 检查磁盘空间是否充足

4. **页面无法访问**
   - 确认服务器已启动
   - 检查防火墙设置
   - 验证访问地址是否正确

### 日志调试

启动时添加详细日志：

```bash
go run main.go 2>&1 | tee app.log
```

## 📝 开发说明

### 技术栈

- **后端**: Go + Gin框架
- **数据库**: SQLite (modernc.org/sqlite)
- **前端**: HTML + CSS + JavaScript + Bootstrap
- **配置**: Viper

### 扩展功能

- 支持更多日志格式
- 添加图表统计
- 实现日志导出功能
- 支持多用户权限管理

## 📄 许可证

本项目采用 MIT 许可证。

## 🤝 贡献

欢迎提交Issue和Pull Request来改进这个项目。 