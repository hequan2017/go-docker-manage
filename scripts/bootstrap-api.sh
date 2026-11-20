#!/usr/bin/env bash
# 部署后初始化:注册默认管理员 + 注册 mock 节点并健康检查(幂等)
set -e
GW="${GATEWAY:-http://127.0.0.1:8888}"

echo "==> waiting gateway"
for i in $(seq 1 60); do
  curl -sf "$GW/api/health" >/dev/null 2>&1 && break
  sleep 1
done

echo "==> ensure admin user"
curl -s -X POST "$GW/api/auth/register" -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"admin123","nickname":"Admin"}' || true
echo

TOKEN=$(curl -s -X POST "$GW/api/auth/login" -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"admin123"}' | sed -n 's/.*"token":"\([^"]*\)".*/\1/p')
if [ -z "$TOKEN" ]; then echo "login failed"; exit 1; fi
echo "==> login ok"

echo "==> ensure mock node"
NODES=$(curl -s "$GW/api/nodes" -H "Authorization: Bearer $TOKEN")
if echo "$NODES" | grep -q 'mock-dockerd'; then
  echo "    mock node already exists"
else
  curl -s -X POST "$GW/api/nodes" -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
    -d '{"name":"mock-node-1","host":"mock-dockerd","dockerApiUrl":"tcp://mock-dockerd:2375","description":"模拟 Docker 节点"}'
  echo
fi

NODE_ID=$(echo "$NODES" | sed -n 's/.*{"id":\([0-9]*\).*mock-dockerd.*/\1/p')
if [ -z "$NODE_ID" ]; then
  NODE_ID=$(curl -s "$GW/api/nodes" -H "Authorization: Bearer $TOKEN" | sed -n 's/.*{"id":\([0-9]*\).*mock-dockerd.*/\1/p')
fi
echo "==> node id: $NODE_ID"

curl -s -X POST "$GW/api/nodes/$NODE_ID/health" -H "Authorization: Bearer $TOKEN"
echo
echo "==> dashboard"
curl -s "$GW/api/dashboard" -H "Authorization: Bearer $TOKEN"
echo
echo "bootstrap done. Web UI: http://<server-ip>/  (admin / admin123)"
