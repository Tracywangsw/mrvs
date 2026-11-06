#!/usr/bin/env bash
#
# MRVS 一键启动脚本：基于 docker-compose.yaml 启动 MySQL 与前后端服务。
#
# 用法：
#   ./start.sh              启动服务（构建镜像并后台运行）
#   ./start.sh stop         停止服务（保留数据库数据）
#   ./start.sh restart      重启服务
#   ./start.sh status       查看容器状态
#   ./start.sh logs         查看并跟随服务日志
#   ./start.sh clean        停止服务并删除数据库数据卷
#
# 可选环境变量：
#   HOST_PORT      宿主机访问端口，默认 8080
#   HEALTH_TIMEOUT 等待服务就绪的超时秒数，默认 120

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

HOST_PORT="${HOST_PORT:-8080}"
HEALTH_TIMEOUT="${HEALTH_TIMEOUT:-120}"
APP_URL="http://127.0.0.1:${HOST_PORT}"
COMPOSE_FILE="docker-compose.yaml"

COMPOSE=()

info() { printf '\033[36m[MRVS]\033[0m %s\n' "$*"; }
error() { printf '\033[31m[MRVS]\033[0m %s\n' "$*" >&2; }
die() {
  error "$*"
  exit 1
}

usage() {
  sed -n '3,15p' "$0" | sed 's/^# \{0,1\}//'
}

detect_compose() {
  command -v docker >/dev/null 2>&1 || die "未找到 Docker，请先安装 Docker Desktop 或 Docker Engine。"
  command -v curl >/dev/null 2>&1 || die "未找到 curl，无法进行健康检查。"

  if docker compose version >/dev/null 2>&1; then
    COMPOSE=(docker compose)
  elif command -v docker-compose >/dev/null 2>&1; then
    COMPOSE=(docker-compose)
  else
    die "未找到 Docker Compose，请安装 Docker Compose 插件。"
  fi

  docker info >/dev/null 2>&1 || die "Docker 守护进程未运行，请先启动 Docker。"

  [ -f "$COMPOSE_FILE" ] || die "未找到 $COMPOSE_FILE。"
}

compose() {
  "${COMPOSE[@]}" -f "$COMPOSE_FILE" "$@"
}

wait_for_health() {
  local start=$SECONDS
  info "等待服务就绪：${APP_URL}/healthz（最长 ${HEALTH_TIMEOUT} 秒）"
  while (( SECONDS - start < HEALTH_TIMEOUT )); do
    if curl -fsS "${APP_URL}/healthz" >/dev/null 2>&1; then
      info "服务已就绪：${APP_URL}"
      return 0
    fi
    sleep 2
  done
  error "服务未在 ${HEALTH_TIMEOUT} 秒内就绪，最近日志如下："
  compose logs --tail=50 app || true
  return 1
}

print_summary() {
  cat <<EOF

服务已启动
  应用地址：${APP_URL}
  健康检查：${APP_URL}/healthz
  API 前缀：${APP_URL}/api/v1

常用命令
  ./start.sh logs     查看服务日志
  ./start.sh status   查看容器状态
  ./start.sh stop     停止服务
EOF
}

cmd_up() {
  detect_compose
  info "构建并启动服务（首次构建可能需要几分钟）…"
  compose up -d --build
  wait_for_health || die "启动失败，请执行 ./start.sh logs 查看日志。"
  print_summary
}

cmd_stop() {
  detect_compose
  info "停止服务…"
  compose down
  info "服务已停止（数据库数据卷已保留）。"
}

cmd_restart() {
  detect_compose
  info "重启服务…"
  compose restart
  wait_for_health || die "重启后服务未就绪，请执行 ./start.sh logs 查看日志。"
  print_summary
}

cmd_status() {
  detect_compose
  compose ps
}

cmd_logs() {
  detect_compose
  compose logs -f --tail=100
}

cmd_clean() {
  detect_compose
  printf '将停止服务并删除数据库数据卷，所有数据不可恢复。确认继续？[y/N] '
  read -r answer
  case "$answer" in
    y | Y | yes | YES)
      compose down --volumes
      info "服务已停止，数据卷已删除。"
      ;;
    *)
      info "已取消。"
      ;;
  esac
}

main() {
  case "${1:-up}" in
    up | start) cmd_up ;;
    stop | down) cmd_stop ;;
    restart) cmd_restart ;;
    status | ps) cmd_status ;;
    logs) cmd_logs ;;
    clean) cmd_clean ;;
    help | -h | --help) usage ;;
    *) die "未知命令：$1（可用：up/stop/restart/status/logs/clean/help）" ;;
  esac
}

main "$@"
