#!/usr/bin/env bash
# 在服务器上：将整个 server 目录拷贝过来后执行
#   cd server/docker && chmod +x deploy.sh && ./deploy.sh
#
# 子命令：
#   ./deploy.sh          构建镜像，按 .env 决定是否 push / 启动
#   ./deploy.sh build    仅构建
#   ./deploy.sh push     构建并推送
#   ./deploy.sh up       构建并 docker compose 启动
#   ./deploy.sh down     停止 compose
#   ./deploy.sh save     构建并导出 tar 包（离线部署）

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVER_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${SCRIPT_DIR}"

load_env() {
  if [[ ! -f .env ]]; then
    return 0
  fi
  set -a
  # 兼容 Windows 拷贝的 CRLF 换行
  # shellcheck disable=SC1090
  source <(sed 's/\r$//' .env)
  set +a
}

load_env

IMAGE_NAME="${IMAGE_NAME:-log-tools-server}"
IMAGE_TAG="${IMAGE_TAG:-latest}"
REGISTRY="${REGISTRY:-}"
PUSH="${PUSH:-true}"
RUN_AFTER_BUILD="${RUN_AFTER_BUILD:-false}"
SAVE_TAR="${SAVE_TAR:-false}"
HOST_PORT="${HOST_PORT:-8080}"

if [[ -n "${REGISTRY}" ]]; then
  IMAGE_FULL="${REGISTRY%/}/${IMAGE_NAME}"
else
  IMAGE_FULL="${IMAGE_NAME}"
fi

export IMAGE_FULL

log() { echo "[deploy] $*"; }

ensure_config() {
  mkdir -p config data/uploads
  if [[ ! -f config/config.yaml ]]; then
    if [[ -f config.example.yaml ]]; then
      cp config.example.yaml config/config.yaml
      log "已从 config.example.yaml 生成 config/config.yaml，请确认 database.host 后重新部署"
    else
      log "错误: 缺少 config/config.yaml"
      exit 1
    fi
  fi
  if grep -qE 'host:[[:space:]]*["'\'']?10\.0\.0\.1' config/config.yaml 2>/dev/null; then
    log "警告: config/config.yaml 中 database.host 仍为示例 10.0.0.1"
    log "       请改为真实 PostgreSQL 地址: cp config.example.yaml config/config.yaml 或手动编辑"
  fi
}

do_build() {
  log "构建镜像 ${IMAGE_FULL}:${IMAGE_TAG} (context: ${SERVER_DIR})"
  docker build -f "${SCRIPT_DIR}/Dockerfile" -t "${IMAGE_FULL}:${IMAGE_TAG}" "${SERVER_DIR}"
  log "构建完成: ${IMAGE_FULL}:${IMAGE_TAG}"
}

do_push() {
  if [[ -z "${REGISTRY}" ]]; then
    log "未设置 REGISTRY，跳过 push"
    return 0
  fi
  log "推送 ${IMAGE_FULL}:${IMAGE_TAG}"
  docker push "${IMAGE_FULL}:${IMAGE_TAG}"
  log "推送完成"
}

do_save() {
  local tar_name="${IMAGE_NAME}-${IMAGE_TAG}.tar"
  log "导出 ${tar_name}"
  docker save -o "${tar_name}" "${IMAGE_FULL}:${IMAGE_TAG}"
  log "已保存: ${SCRIPT_DIR}/${tar_name}"
  log "离线加载: docker load -i ${tar_name}"
}

do_up() {
  ensure_config
  log "启动服务 (compose)..."
  docker compose up -d --build
  log "服务已启动: http://<host>:${HOST_PORT}"
  docker compose ps
}

do_down() {
  docker compose down
  log "已停止"
}

CMD="${1:-all}"

case "${CMD}" in
  build)
    do_build
    ;;
  push)
    do_build
    do_push
    ;;
  save)
    do_build
    do_save
    ;;
  up)
    do_build
    if [[ "${PUSH}" == "true" && -n "${REGISTRY}" ]]; then
      do_push || true
    fi
    do_up
    ;;
  down)
    do_down
    ;;
  all|"")
    do_build
    if [[ "${PUSH}" == "true" && -n "${REGISTRY}" ]]; then
      do_push
    fi
    if [[ "${SAVE_TAR}" == "true" ]]; then
      do_save
    fi
    if [[ "${RUN_AFTER_BUILD}" == "true" ]]; then
      do_up
    else
      log "完成。启动服务: ./deploy.sh up"
      log "或: docker run -d -p ${HOST_PORT}:8080 -v \$(pwd)/config/config.yaml:/app/config/config.yaml:ro -v \$(pwd)/data:/app/data ${IMAGE_FULL}:${IMAGE_TAG}"
    fi
    ;;
  *)
    echo "用法: $0 [build|push|save|up|down|all]"
    exit 1
    ;;
esac
