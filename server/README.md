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

## 运行

```bash
cd server
go mod tidy
go run .
```

默认监听 `0.0.0.0:8080`。

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
| POST | /api/jira/issues/:key/attachments | 列出 Jira 附件 |
| POST | /api/jira/import | 导入选中 Jira 附件 |
