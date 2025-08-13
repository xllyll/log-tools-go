# 日志分析工具

## 项目概述
开发一个基于Go语言的本地日志分析系统，提供Web界面进行日志文件上传、解析、过滤和可视化展示。

## 功能需求
1. **Web界面**: 运行main.go后打开 http://127.0.0.1:4080
2. **文件上传**: 支持上传txt、log文件和压缩包，自动解压
3. **日志解析**: 根据日志级别显示不同颜色
4. **过滤功能**: 通过配置文件筛选日志
5. **本地存储**: 上传的文件保存到本地
6. **打包部署**: 可打包成exe文件，无需网络连接

## 技术架构

### 后端技术栈
- **语言**: Go 1.24
- **Web框架**: Gin (轻量级HTTP框架)
- **模板引擎**: HTML模板
- **文件处理**: archive/zip, compress/gzip
- **日志解析**: 正则表达式
- **配置管理**: Viper

### 前端技术栈
- **HTML5**: 页面结构
- **CSS3**: 样式和颜色主题
- **JavaScript**: 交互逻辑
- **Bootstrap**: UI组件库

## 开发步骤

### 第一步：项目初始化和依赖管理
```bash
# 初始化Go模块
go mod init log-tools-go

# 添加依赖
go get github.com/gin-gonic/gin
go get github.com/spf13/viper
go get github.com/gorilla/websocket
```

### 第二步：项目结构设计
```
log-tools-go/
├── main.go                 # 主程序入口
├── go.mod                  # Go模块文件
├── go.sum                  # 依赖校验文件
├── config/
│   ├── config.yaml        # 配置文件
│   └── config.go          # 配置管理
├── internal/
│   ├── handler/           # HTTP处理器
│   │   ├── upload.go      # 文件上传处理
│   │   ├── log.go         # 日志展示处理
│   │   └── filter.go      # 日志过滤处理
│   ├── service/           # 业务逻辑层
│   │   ├── parser.go      # 日志解析服务
│   │   ├── storage.go     # 文件存储服务
│   │   └── filter.go      # 过滤服务
│   └── model/             # 数据模型
│       └── log.go         # 日志数据结构
├── web/                   # 前端资源
│   ├── static/            # 静态文件
│   │   ├── css/
│   │   ├── js/
│   │   └── images/
│   └── templates/         # HTML模板
│       ├── index.html     # 主页面
│       └── components/    # 组件模板
├── uploads/               # 上传文件存储目录
├── logs/                  # 解析后的日志存储
└── build/                 # 构建输出目录
```

### 第三步：核心功能实现

#### 3.1 主程序入口 (main.go)
- 初始化Gin Web服务器
- 配置路由和中间件
- 启动HTTP服务在4080端口

#### 3.2 文件上传功能
- 支持拖拽上传
- 文件类型验证 (txt, log, zip, gz)
- 自动解压缩处理
- 文件存储到本地uploads目录

#### 3.3 日志解析功能
- 正则表达式匹配日志格式
- 提取时间戳、日志级别、消息内容
- 支持多种日志格式 (JSON, 标准格式等)
- 日志级别颜色映射

#### 3.4 过滤和搜索功能
- 基于日志级别的过滤
- 关键词搜索
- 时间范围筛选
- 配置文件驱动的过滤规则

#### 3.5 Web界面设计
- 响应式布局
- 文件上传区域
- 日志展示表格
- 过滤控制面板
- 实时搜索功能

### 第四步：配置文件设计

#### config.yaml 示例
```yaml
server:
  port: 4080
  host: "127.0.0.1"

storage:
  upload_dir: "./uploads"
  log_dir: "./logs"
  max_file_size: 100MB

log_levels:
  ERROR: "#dc3545"    # 红色
  WARN: "#ffc107"     # 黄色
  INFO: "#17a2b8"     # 蓝色
  DEBUG: "#6c757d"    # 灰色

filters:
  default_levels: ["ERROR", "WARN", "INFO"]
  exclude_patterns: []
  include_patterns: []
```

### 第五步：前端界面实现

#### 主要页面组件
1. **文件上传区域**
   - 拖拽上传支持
   - 文件类型提示
   - 上传进度显示

2. **日志展示区域**
   - 表格形式展示
   - 分页功能
   - 排序功能
   - 颜色编码

3. **过滤控制面板**
   - 日志级别选择
   - 关键词搜索
   - 时间范围选择
   - 高级过滤选项

### 第六步：构建和打包

#### 本地开发运行
```bash
# 开发模式运行
go run main.go

# 或者构建后运行
go build -o log-analyzer.exe
./log-analyzer.exe
```

#### 打包成独立exe
```bash
# 使用go build打包
go build -ldflags="-s -w" -o log-analyzer.exe

# 或使用upx压缩 (可选)
upx --best log-analyzer.exe
```

### 第七步：测试和优化

#### 功能测试
- 文件上传测试
- 日志解析准确性测试
- 过滤功能测试
- 界面响应性测试

#### 性能优化
- 大文件处理优化
- 内存使用优化
- 并发处理优化

## 部署说明

### 开发环境
1. 确保Go 1.24+已安装
2. 克隆项目到本地
3. 运行 `go mod tidy` 安装依赖
4. 执行 `go run main.go` 启动服务
5. 访问 http://127.0.0.1:4080

### 生产环境
1. 构建exe文件: `go build -o log-analyzer.exe`
2. 复制配置文件到exe同目录
3. 创建必要的目录结构
4. 运行exe文件即可使用

## 注意事项

1. **安全性**: 文件上传需要验证文件类型和大小
2. **性能**: 大文件处理需要考虑内存使用
3. **兼容性**: 支持Windows、Linux、macOS
4. **数据持久化**: 上传的文件和解析结果需要持久化存储
5. **用户体验**: 提供友好的错误提示和加载状态

## 扩展功能 (可选)

1. **日志统计**: 按级别、时间统计日志数量
2. **导出功能**: 支持导出过滤后的日志
3. **多文件对比**: 支持多个日志文件对比分析
4. **实时监控**: WebSocket实时更新日志
5. **插件系统**: 支持自定义日志格式解析器 