# go-docker-manage

基于 **go-zero 微服务 + Nacos** 的 Docker 管理系统。各服务通过官方 Docker SDK 对接 Docker Engine API，开发阶段由 `mock-dockerd` 模拟 Docker daemon。要切到真实 Docker，只需把节点的 Docker API 地址改成真实地址，业务代码不用改。

> 当前进度：gateway、user-rpc、node-rpc、container-rpc、image-rpc、mock-dockerd 已实现；Web UI 和部署脚本还没开发（下图虚线部分）。

## 架构图

```mermaid
flowchart TB
    browser([浏览器])

    web["web（规划中）<br/>nginx + Vue3 + Element Plus<br/>:80"]
    gateway["gateway<br/>go-zero REST + JWT 鉴权<br/>:8888"]

    nacos[("Nacos<br/>注册中心 + 配置中心<br/>:8848")]

    subgraph rpc [gRPC 微服务]
        user["user-rpc :9000<br/>注册 / 登录 / 用户信息"]
        node["node-rpc :9001<br/>节点 CRUD / 健康检查"]
        container["container-rpc :9002<br/>容器列表 / 启停 / 创建 / 日志 / stats"]
        image["image-rpc :9003<br/>镜像列表 / 拉取 / 删除"]
    end

    userdb[("SQLite<br/>user.db")]
    nodedb[("SQLite<br/>node.db")]

    subgraph engine [Docker Engine API]
        mock["mock-dockerd :2375<br/>Docker API 模拟器"]
        real["真实 dockerd<br/>tcp://host:2375"]
    end

    browser -->|HTTP| web
    web -->|/api 代理| gateway
    gateway -->|gRPC| user
    gateway -->|gRPC| node
    gateway -->|gRPC| container
    gateway -->|gRPC| image

    user --- userdb
    node --- nodedb

    container -->|GetNode 查 DockerApiUrl| node
    image -->|GetNode 查 DockerApiUrl| node
    container -->|Docker SDK| mock
    image -->|Docker SDK| mock
    container -.->|改节点地址即可切换| real
    node -->|"GET /_ping 健康检查"| mock

    user -.->|注册 / 拉配置| nacos
    node -.->|注册 / 拉配置| nacos
    container -.->|注册 / 发现 nacos://node-rpc| nacos
    image -.->|注册 / 发现| nacos
    gateway -.->|服务发现| nacos

    classDef todo stroke-dasharray: 5 5,fill:#f6f6f6,color:#888
    class web,real todo
```

### 调用链路示例：查看某节点的容器列表

```mermaid
sequenceDiagram
    participant G as gateway
    participant C as container-rpc
    participant N as node-rpc
    participant D as mock-dockerd / dockerd
    G->>C: ListContainers(node_id)
    C->>N: GetNode(id)
    N-->>C: NodeInfo{DockerApiUrl}
    C->>D: GET /v1.4x/containers/json（Docker SDK）
    D-->>C: 容器列表
    C-->>G: ListContainersReply
```

## 目录结构

```
app/
  user/rpc/        用户服务（SQLite，bcrypt 密码）
  node/rpc/        节点服务（SQLite，/_ping 健康检查）
  container/rpc/   容器服务（无状态，实时查 Docker API）
  image/rpc/       镜像服务（列表 / 拉取 / 删除）
  gateway/         API 网关（REST + JWT，HTTP → gRPC）
pkg/
  nacosx/          go-zero ⇄ Nacos 适配：服务注册、nacos:// gRPC resolver、配置中心拉取
  dockerx/         Docker SDK 客户端（按地址缓存复用）
  errorx/          统一业务错误码
  zrpcx/           zrpc 客户端构造：启用 Nacos 走服务发现，否则直连
examples/greeter/  最小 zrpc 微服务示例（服务端 + 客户端）
mock-dockerd/      Docker Engine API 模拟器（容器/镜像/日志/stats，可持久化为 JSON）
```

## 快速开始

```bash
go mod tidy          # 首次构建前执行，补齐依赖
go build ./...
go test ./...

# 1. 启动模拟 Docker daemon
go run ./mock-dockerd -addr :2375 -data state.json

# 2. 启动各服务（配置路径是相对路径，需在服务目录内运行）
cd app/user/rpc      && go run . -f etc/user.yaml
cd app/node/rpc      && go run . -f etc/node.yaml
cd app/container/rpc && go run . -f etc/container.yaml
cd app/image/rpc     && go run . -f etc/image.yaml
cd app/gateway       && go run . -f etc/gateway.yaml   # HTTP :8888
```

网关主要接口（除注册 / 登录 / health 外都需要 `Authorization: Bearer <token>`）：

| 接口 | 说明 |
|---|---|
| `POST /api/auth/register`、`POST /api/auth/login` | 注册 / 登录（返回 JWT） |
| `GET/POST /api/nodes`、`DELETE /api/nodes/:nodeId`、`POST /api/nodes/:nodeId/health` | 节点管理 |
| `/api/nodes/:nodeId/containers[/:containerId[/:action\|/logs\|/stats]]` | 容器管理 |
| `/api/nodes/:nodeId/images`、`POST /api/nodes/:nodeId/images/pull` | 镜像管理 |
| `GET /api/dashboard` | 节点 / 容器 / 镜像统计 |

mock-dockerd 兼容 docker CLI，可以直接用来验证：`docker -H tcp://127.0.0.1:2375 ps -a`。

## 配置与 Nacos

- 本地的 `etc/<服务>.yaml` 只放启动信息（Name、ListenOn、Nacos 地址）。
- `Nacos.Host` 为空（默认）时不使用 Nacos，服务之间按 `Endpoints` 直连，适合本地调试。
- `Nacos.Host` 填上地址后：
  - 启动时从配置中心拉取与服务同名的 dataId（如 `node-rpc`），覆盖本地配置；
  - 服务实例自动注册到 Nacos；
  - 客户端用 `nacos://<服务名>` 做服务发现。

## 重新生成 protobuf 代码

在仓库根目录执行。需要 [protoc](https://github.com/protocolbuffers/protobuf/releases)（本地放在 `.tools/`，该目录不提交）以及插件 `go install google.golang.org/protobuf/cmd/protoc-gen-go@latest` 和 `google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest`：

```bash
PATH="$HOME/go/bin:$PATH" .tools/bin/protoc.exe -I app/<svc>/rpc -I .tools/include \
  --go_out=. --go_opt=module=github.com/hequan2017/go-docker-manage \
  --go-grpc_out=. --go-grpc_opt=module=github.com/hequan2017/go-docker-manage \
  app/<svc>/rpc/<svc>.proto
```

## License

见 [LICENSE](LICENSE)。
