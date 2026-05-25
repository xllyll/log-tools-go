# log-tools-server Docker 部署

将整个 `server` 目录拷贝到目标机器后，在 `server/docker` 下执行脚本即可完成镜像构建与发布。

**PostgreSQL 需单独部署**，在 `config/config.yaml` 中配置连接信息；compose 仅启动应用容器。

## 目录说明

| 文件 | 说明 |
|------|------|
| `Dockerfile` | 多阶段构建，产物 `/app/log-tools-server` |
| `docker-compose.yml` | 仅 log-tools-server 应用 |
| `config.example.yaml` | 配置模板，首次部署自动复制为 `config/config.yaml` |
| `.env.example` | 镜像名、仓库地址等，复制为 `.env` |
| `deploy.sh` | Linux 一键脚本（构建 / push / 启动） |
| `deploy.ps1` | Windows 构建脚本 |

## 快速开始（Linux 服务器）

```bash
cd server/docker
cp .env.example .env
cp config.example.yaml config/config.yaml
# 编辑 config/config.yaml：database.host 填外部 PG 地址（容器需能访问）
# 编辑 .env（镜像仓库等）

chmod +x deploy.sh

./deploy.sh build    # 仅构建镜像
./deploy.sh push     # 构建并推送（需 docker login）
./deploy.sh up       # 构建并启动应用
./deploy.sh down     # 停止
```

## 环境变量（`.env`）

```bash
IMAGE_NAME=log-tools-server
IMAGE_TAG=1.0.0
REGISTRY=registry.example.com/your-team   # 留空则不 push
PUSH=true
RUN_AFTER_BUILD=false
SAVE_TAR=false
HOST_PORT=8080
```

## 数据库说明

- 使用**已部署的 PostgreSQL**，在 `config/config.yaml` 配置 `database.host` / `password` 等
- 若 PG 在宿主机上，Linux 可用 `host.docker.internal` 或宿主机内网 IP（不要用 `127.0.0.1`，容器内指向自身）
- 确保 PG 的 `pg_hba.conf` 放行 Docker 网段或应用服务器 IP

## 无镜像仓库（离线）

```bash
./deploy.sh save
docker load -i log-tools-server-latest.tar
docker run -d --name log-tools \
  -p 8080:8080 \
  -v $(pwd)/config/config.yaml:/app/config/config.yaml:ro \
  -v $(pwd)/data:/app/data \
  log-tools-server:latest
```

## 数据持久化

- 上传文件：`docker/data/uploads` → 容器 `/app/data/uploads`
- 配置：`docker/config/config.yaml` → 容器 `/app/config/config.yaml`

## 构建说明

- Docker 构建上下文为 **`server/`** 根目录（非 `docker/`）
- 容器内通过环境变量 `CONFIG_PATH` 指定配置，默认 `/app/config/config.yaml`
