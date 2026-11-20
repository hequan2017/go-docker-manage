#!/bin/sh
# 将 configs/*.yaml 导入 Nacos 配置中心(幂等,覆盖式)
set -e
NACOS="${NACOS_ADDR:-http://nacos:8848}"
GROUP="${NACOS_GROUP:-DEFAULT_GROUP}"

# 等待 Nacos 就绪
i=0
until curl -sf "$NACOS/nacos/v1/console/health/readiness" >/dev/null 2>&1; do
  i=$((i + 1))
  [ "$i" -ge 60 ] && echo "nacos not ready after 60s" && exit 1
  sleep 1
done

for f in /configs/*.yaml; do
  dataId="$(basename "$f")"
  echo "publishing $dataId (group=$GROUP)"
  curl -sf -X POST "$NACOS/nacos/v1/cs/configs" \
    --data-urlencode "dataId=$dataId" \
    --data-urlencode "group=$GROUP" \
    --data-urlencode "type=yaml" \
    --data-urlencode "content@/$f" >/dev/null
  echo "  -> ok"
done
echo "nacos configs initialized."
