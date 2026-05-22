# log-tools web

Vue 3 + Vite + Element Plus 前端。

## 功能

- 浏览器设备 ID 自动缓存（`localStorage`）
- 上传 `.log/.txt/.zip/.rar/.7z`
- 文件列表与日志查看
- 点击行展开前后 10 条上下文
- 关键词 / 正则搜索
- 场景配置（本地 JSON + 可选同步服务器）
- Jira 附件拉取与导入

## 运行

```bash
cd web
npm install
npm run dev
```

开发服务器 `http://localhost:5173`，API 通过 Vite 代理到 `http://127.0.0.1:8080`。
