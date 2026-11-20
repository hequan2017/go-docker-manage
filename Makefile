.PHONY: build test mock web deploy local-up clean

# 构建全部服务二进制(本地开发用)
build:
	go build -o bin/mock-dockerd.exe ./mock-dockerd
	go build -o bin/user-rpc.exe ./app/user/rpc
	go build -o bin/node-rpc.exe ./app/node/rpc
	go build -o bin/container-rpc.exe ./app/container/rpc
	go build -o bin/image-rpc.exe ./app/image/rpc
	go build -o bin/gateway.exe ./app/gateway

test:
	go test ./...

# 本地直连模式(不依赖 Nacos)依次启动:mock -> 各 rpc -> gateway
# Windows 下建议分终端执行,配置已内置 127.0.0.1 直连
local-up:
	@echo "请分进程执行(见 README「本地直连模式」):"
	@echo "  mock-dockerd:  bin/mock-dockerd.exe -addr :2375"
	@echo "  user-rpc:      (cd app/user/rpc && ../../bin/user-rpc.exe -f etc/user.yaml)"
	@echo "  node-rpc:      (cd app/node/rpc && ../../bin/node-rpc.exe -f etc/node.yaml)"
	@echo "  container-rpc: (cd app/container/rpc && ../../bin/container-rpc.exe -f etc/container.yaml)"
	@echo "  image-rpc:     (cd app/image/rpc && ../../bin/image-rpc.exe -f etc/image.yaml)"
	@echo "  gateway:       (cd app/gateway && ../bin/gateway.exe -f etc/gateway.yaml)"

web:
	cd web && npm install && npm run build

# 部署到 192.168.112.138(rsync + docker compose)
deploy:
	bash scripts/deploy.sh

clean:
	rm -rf bin web/dist
