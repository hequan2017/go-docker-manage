#!/usr/bin/env bash
# 一键部署到 192.168.112.138:
#   ./scripts/deploy.sh [user@host]
# 流程:rsync 源码 -> 远程 docker compose build && up -> API 引导(建节点/健康检查)
set -e

SERVER="${1:-root@192.168.112.138}"
REMOTE_DIR="/opt/go-docker-manage"

echo "==> [1/4] sync source to $SERVER:$REMOTE_DIR"
if command -v rsync >/dev/null 2>&1; then
  rsync -az --delete \
    --exclude .git --exclude .tools --exclude bin \
    --exclude web/node_modules --exclude web/dist \
    --exclude 'app/*/rpc/data' --exclude '*.db' \
    ./ "$SERVER:$REMOTE_DIR/"
else
  # 无 rsync(如 Windows)时用 tar 管道
  ssh "$SERVER" "mkdir -p $REMOTE_DIR"
  tar czf - \
    --exclude=.git --exclude=.tools --exclude=bin \
    --exclude=web/node_modules --exclude=web/dist \
    --exclude='app/*/rpc/data' --exclude='*.db' --exclude='*.exe' \
    . | ssh "$SERVER" "tar xzf - -C $REMOTE_DIR"
fi

echo "==> [2/4] build images (on server)"
ssh "$SERVER" "cd $REMOTE_DIR && docker compose -f deploy/docker-compose.yml build --pull"

echo "==> [3/4] up"
ssh "$SERVER" "cd $REMOTE_DIR && docker compose -f deploy/docker-compose.yml up -d"

echo "==> [4/4] bootstrap"
ssh "$SERVER" "cd $REMOTE_DIR && GATEWAY=http://127.0.0.1:8888 bash scripts/bootstrap-api.sh"

echo
echo "deploy finished:"
echo "  Web UI : http://192.168.112.138:8180/"
echo "  API    : http://192.168.112.138:8888/api/health"
echo "  Nacos  : http://192.168.112.138:8848/nacos  (nacos/nacos)"
