# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

Docker management system built as go-zero gRPC microservices with Nacos (service registry + config center). Docker Engine is reached via the official Docker SDK; during development it points at `mock-dockerd`, a Docker Engine API simulator. Switching to a real daemon is purely a data change (a node's Docker API URL), not a code change.

Planned but not yet built: a Vue3 `web/` UI (nginx, proxies `/api` to the gateway), `deploy/` (docker-compose with Nacos standalone, multi-stage Dockerfiles, nacos-init scripts that import each service's dataId), and `scripts/deploy.sh`. The repo uses a **single root `go.mod`** (`github.com/hequan2017/go-docker-manage`), not a `go.work`. Code comments and user-facing messages are in Chinese; keep that style.

`examples/greeter/` is a minimal standalone zrpc server/client used as a teaching example. It is not part of the system.

## Commands

```bash
go build ./...
go test ./...
go test ./mock-dockerd -run TestContainerLifecycle -v   # single test
go vet ./...
~/go/bin/staticcheck.exe ./...                # staticcheck is installed locally

# Run a service: the -f default is relative (etc/<svc>.yaml), so run it from the service dir
cd app/node/rpc && go run . -f etc/node.yaml
go run ./mock-dockerd -addr :2375 -data state.json   # -data empty = in-memory only
```

Regenerate protobuf code from the repo root. `protoc` is kept locally in `.tools/bin/` (gitignored, so it may be missing on a fresh clone) and the plugins in `~/go/bin`. This command reproduces the committed `pb/` files byte-for-byte:

```bash
PATH="$HOME/go/bin:$PATH" .tools/bin/protoc.exe -I app/<svc>/rpc -I .tools/include \
  --go_out=. --go_opt=module=github.com/hequan2017/go-docker-manage \
  --go-grpc_out=. --go-grpc_opt=module=github.com/hequan2017/go-docker-manage \
  app/<svc>/rpc/<svc>.proto
```

goctl is not installed. Service skeletons follow the goctl layout (`internal/{config,logic,server,svc}`), but they are written by hand.

## Architecture

```
gateway :8888 (REST, JWT) ──gRPC──▶ user-rpc :9000  (SQLite)
                                 node-rpc :9001  (SQLite; node CRUD + /_ping health check)
                                 container-rpc :9002 ─┐ stateless: look up node via node-rpc,
                                 image-rpc :9003 ─────┘ then call Docker SDK at node.DockerApiUrl
                                                          └─▶ mock-dockerd :2375 (or real dockerd)
```

### Nacos integration (`pkg/nacosx`) and the "Nacos optional" pattern
Each service's `main.go` repeats the same bootstrap. Follow it for new services:
1. `conf.MustLoad` the local `etc/<svc>.yaml`. It holds only bootstrap info (Name, ListenOn, Nacos address).
2. `nacosx.MustLoadRemote(c.Nacos, c.Name, &c)` pulls the Nacos config dataId named after the service (`node-rpc`, etc.) and overlays it on the local config.
3. After `zrpc.MustNewServer`, if Nacos is enabled, register the instance with `RegisterService` (local IP + port parsed from ListenOn) and `defer DeregisterService`.

`nacosx.Config.Empty()` (Host == "") turns all Nacos features off. The local yaml files ship with `Host: ""`, so services run standalone on direct endpoints by default.

Upstream clients are built with `pkg/zrpcx.NewClient(upstream, nacosCfg)`. With Nacos enabled it registers the `nacos://` resolver once and dials `Target: nacos://<service-name>`; otherwise it dials `Endpoints`. Each upstream is configured as a `zrpcx.Upstream` block in yaml (see `app/gateway/etc/gateway.yaml`). The resolver is hand-written against nacos-sdk-go/v2. It does not use the zeromicro/x adapters.

### Docker access
- `pkg/dockerx.New(addr)` returns a Docker SDK client cached per address, with API version negotiation.
- container/image logic first calls `NodeCli.GetNode(nodeId)` to get `DockerApiUrl`, then calls the SDK.
- RPC services return gRPC `status` codes. The gateway (`app/gateway/gateway.go`) maps them, plus `errorx.StatusError`, to `{code, msg}` JSON bodies with an HTTP status derived from the `pkg/errorx` business codes.

### Gateway (`app/gateway`)
go-zero `rest` server. Routes are registered by hand in `internal/handler/handlers.go`, not generated from a `.api` file. `/api/health` and `/api/auth/*` are public; everything else is behind JWT (`Auth.AccessSecret`). Resources are nested by node: `/api/nodes/:nodeId/containers/...` and `/api/nodes/:nodeId/images/...`. `/api/dashboard` aggregates counts across services.

### mock-dockerd (`package main`, stdlib `net/http` only)
- `router.go` strips the `/vX.Y/` version prefix the SDK adds, then dispatches by hand on `/containers/{id}/{action}` and `/images/{name}`.
- Errors must use the Docker format `{"message": ...}`. The SDK maps status codes to `errdefs` from it.
- State lives in an in-memory `store`, guarded by `s.mu`. Methods ending in `Locked` expect the caller to hold the lock.
- `seed.go` provides preset containers/images. `persist.go` writes a JSON snapshot (logs are not persisted). `logs.go` emits Docker's 8-byte multiplexed stream format, so `stdcopy` works.
- `server_test.go` runs the real Docker SDK against `httptest` to prove compatibility. Add a test like these whenever you add an endpoint.

### Persistence
user-rpc and node-rpc each own a private SQLite file (`DB.Path`, default `data/<svc>.db`), opened with GORM + `glebarez/sqlite`. That driver is pure Go (no CGO), which keeps cross-compiling to linux/amd64 simple. Schema is managed with `AutoMigrate` in the service context. Passwords are stored as bcrypt hashes.
