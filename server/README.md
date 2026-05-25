# log-tools server

Go + Gin + PostgreSQL 日志分析服务端。

## 环境

- Go 1.22+
- PostgreSQL 14+

## 数据库

```sql
CREATE DATABASE log_tools;
```

修改 `config/config.yaml` 中的数据库连接信息。

上传日志默认保留 **30 天**，服务启动后会立即执行一次清理，之后每 24 小时扫描一次（删除库记录、解析行及磁盘上的源文件/解压目录）。可在 `storage.retention_days` 调整天数，设为 `0` 关闭；`storage.cleanup_interval_hours` 调整间隔。

## 运行

```bash
cd server
go mod tidy
go run .
```

默认监听 `0.0.0.0:8080`。

## Docker 部署

PostgreSQL 需单独部署，compose 仅包含应用。详见 [docker/README.md](docker/README.md)。

```bash
cd server/docker
cp .env.example .env && cp config.example.yaml config/config.yaml
# 编辑 config/config.yaml 中的 database 连接外部 PG
chmod +x deploy.sh && ./deploy.sh up
```

## API 说明

所有 `/api/*` 接口需请求头 `X-Device-ID`（浏览器设备标识）。

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/upload | 上传日志（异步入库） |
| GET | /api/files | 当前设备文件列表 |
| DELETE | /api/files/:id | 删除文件 |
| POST | /api/logs/query | 查询日志（关键词 AND / 场景 OR） |
| GET | /api/logs/context | 上下文行（前后 N 条） |
| POST | /api/scenes | 保存场景配置 |
| GET | /api/scene-library | 场景库列表（全员共享） |
| POST | /api/scene-library | 上传场景包到场景库 |
| GET | /api/scene-library/:id | 获取场景包详情 |
| DELETE | /api/scene-library/:id | 删除自己上传的场景包 |
| GET | /api/jira/issues/:key/attachments | 列出 Issue 日志附件（凭据在服务端） |
| POST | /api/jira/import | 导入选中 Jira 附件 |

## Jira 同步

在 `config/config.yaml` 中配置（前端只需填写 Issue Key）：

```yaml
jira:
  enabled: true
  base_url: "https://jira.example.com"
  email: "your-email@example.com"
  api_token: "your-api-token"
```

修改后需重启服务。`enabled: false` 时相关接口会返回未启用提示。
