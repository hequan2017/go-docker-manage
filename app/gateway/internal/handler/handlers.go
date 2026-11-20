// Handler 层:参数解析 → 调用 logic → 统一响应体输出
package handler

import (
	"net/http"

	"github.com/hequan2017/go-docker-manage/app/gateway/internal/logic"
	"github.com/hequan2017/go-docker-manage/app/gateway/internal/svc"
	"github.com/hequan2017/go-docker-manage/app/gateway/internal/types"
	"github.com/hequan2017/go-docker-manage/pkg/errorx"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// RegisterHandlers 注册全部路由(公开路由 + JWT 路由)
func RegisterHandlers(server *rest.Server, svcCtx *svc.ServiceContext, jwtSecret string) {
	// 公开路由
	server.AddRoutes([]rest.Route{
		{Method: http.MethodGet, Path: "/api/health", Handler: healthHandler()},
		{Method: http.MethodPost, Path: "/api/auth/register", Handler: registerHandler(svcCtx)},
		{Method: http.MethodPost, Path: "/api/auth/login", Handler: loginHandler(svcCtx)},
	})

	// 需登录路由
	server.AddRoutes([]rest.Route{
		{Method: http.MethodGet, Path: "/api/user/info", Handler: userInfoHandler(svcCtx)},

		{Method: http.MethodGet, Path: "/api/nodes", Handler: nodeListHandler(svcCtx)},
		{Method: http.MethodPost, Path: "/api/nodes", Handler: nodeCreateHandler(svcCtx)},
		{Method: http.MethodPost, Path: "/api/nodes/:nodeId/health", Handler: nodeHealthHandler(svcCtx)},
		{Method: http.MethodDelete, Path: "/api/nodes/:nodeId", Handler: nodeDeleteHandler(svcCtx)},

		{Method: http.MethodGet, Path: "/api/nodes/:nodeId/containers", Handler: containerListHandler(svcCtx)},
		{Method: http.MethodPost, Path: "/api/nodes/:nodeId/containers", Handler: containerCreateHandler(svcCtx)},
		{Method: http.MethodPost, Path: "/api/nodes/:nodeId/containers/:containerId/:action", Handler: containerActionHandler(svcCtx)},
		{Method: http.MethodDelete, Path: "/api/nodes/:nodeId/containers/:containerId", Handler: containerRemoveHandler(svcCtx)},
		{Method: http.MethodGet, Path: "/api/nodes/:nodeId/containers/:containerId/logs", Handler: containerLogsHandler(svcCtx)},
		{Method: http.MethodGet, Path: "/api/nodes/:nodeId/containers/:containerId/stats", Handler: containerStatsHandler(svcCtx)},

		{Method: http.MethodGet, Path: "/api/nodes/:nodeId/images", Handler: imageListHandler(svcCtx)},
		{Method: http.MethodPost, Path: "/api/nodes/:nodeId/images/pull", Handler: imagePullHandler(svcCtx)},
		{Method: http.MethodDelete, Path: "/api/nodes/:nodeId/images", Handler: imageRemoveHandler(svcCtx)},

		{Method: http.MethodGet, Path: "/api/dashboard", Handler: dashboardHandler(svcCtx)},
	}, rest.WithJwt(jwtSecret))
}

// ok / fail 统一输出
func ok(w http.ResponseWriter, data any) {
	httpx.OkJson(w, types.Body{Code: errorx.CodeOK, Msg: "ok", Data: data})
}

func fail(w http.ResponseWriter, err error) {
	httpx.Error(w, err)
}

func parse[T any](w http.ResponseWriter, r *http.Request) (*T, bool) {
	var req T
	if err := httpx.Parse(r, &req); err != nil {
		httpx.Error(w, errorx.New(errorx.CodeInvalidParam, err.Error()))
		return nil, false
	}
	return &req, true
}

// ---------- system ----------

func healthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) { ok(w, "up") }
}

// ---------- auth ----------

func registerHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, okReq := parse[types.RegisterReq](w, r)
		if !okReq {
			return
		}
		l := logic.NewAuthLogic(r.Context(), svcCtx)
		resp, err := l.Register(req)
		if err != nil {
			fail(w, err)
			return
		}
		ok(w, resp)
	}
}

func loginHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, okReq := parse[types.LoginReq](w, r)
		if !okReq {
			return
		}
		l := logic.NewAuthLogic(r.Context(), svcCtx)
		resp, err := l.Login(req)
		if err != nil {
			fail(w, err)
			return
		}
		ok(w, resp)
	}
}

func userInfoHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logic.NewAuthLogic(r.Context(), svcCtx)
		resp, err := l.UserInfo()
		if err != nil {
			fail(w, err)
			return
		}
		ok(w, resp)
	}
}

// ---------- node ----------

func nodeListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logic.NewNodeLogic(r.Context(), svcCtx)
		resp, err := l.List()
		if err != nil {
			fail(w, err)
			return
		}
		ok(w, resp)
	}
}

func nodeCreateHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, okReq := parse[types.CreateNodeReq](w, r)
		if !okReq {
			return
		}
		l := logic.NewNodeLogic(r.Context(), svcCtx)
		resp, err := l.Create(req)
		if err != nil {
			fail(w, err)
			return
		}
		ok(w, resp)
	}
}

func nodeHealthHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, okReq := parse[types.NodeIDPathReq](w, r)
		if !okReq {
			return
		}
		l := logic.NewNodeLogic(r.Context(), svcCtx)
		resp, err := l.Health(req.NodeId)
		if err != nil {
			fail(w, err)
			return
		}
		ok(w, resp)
	}
}

func nodeDeleteHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, okReq := parse[types.NodeIDPathReq](w, r)
		if !okReq {
			return
		}
		l := logic.NewNodeLogic(r.Context(), svcCtx)
		if err := l.Delete(req.NodeId); err != nil {
			fail(w, err)
			return
		}
		ok(w, nil)
	}
}

// ---------- container ----------

func containerListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, okReq := parse[types.ContainerListReq](w, r)
		if !okReq {
			return
		}
		l := logic.NewContainerLogic(r.Context(), svcCtx)
		resp, err := l.List(req.NodeId, req.All)
		if err != nil {
			fail(w, err)
			return
		}
		ok(w, resp)
	}
}

func containerCreateHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, okReq := parse[types.CreateContainerReq](w, r)
		if !okReq {
			return
		}
		l := logic.NewContainerLogic(r.Context(), svcCtx)
		resp, err := l.Create(req)
		if err != nil {
			fail(w, err)
			return
		}
		ok(w, resp)
	}
}

func containerActionHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, okReq := parse[types.ContainerActionReq](w, r)
		if !okReq {
			return
		}
		l := logic.NewContainerLogic(r.Context(), svcCtx)
		msg, err := l.Action(req.NodeId, req.ContainerId, req.Action)
		if err != nil {
			fail(w, err)
			return
		}
		ok(w, msg)
	}
}

func containerRemoveHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, okReq := parse[types.RemoveContainerReq](w, r)
		if !okReq {
			return
		}
		l := logic.NewContainerLogic(r.Context(), svcCtx)
		if err := l.Remove(req.NodeId, req.ContainerId, req.Force); err != nil {
			fail(w, err)
			return
		}
		ok(w, nil)
	}
}

func containerLogsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, okReq := parse[types.ContainerLogsReq](w, r)
		if !okReq {
			return
		}
		l := logic.NewContainerLogic(r.Context(), svcCtx)
		logs, err := l.Logs(req.NodeId, req.ContainerId, req.Tail, req.Timestamps)
		if err != nil {
			fail(w, err)
			return
		}
		ok(w, logs)
	}
}

func containerStatsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, okReq := parse[types.ContainerStatsReq](w, r)
		if !okReq {
			return
		}
		l := logic.NewContainerLogic(r.Context(), svcCtx)
		resp, err := l.Stats(req.NodeId, req.ContainerId)
		if err != nil {
			fail(w, err)
			return
		}
		ok(w, resp)
	}
}

// ---------- image ----------

func imageListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, okReq := parse[types.NodeIDPathReq](w, r)
		if !okReq {
			return
		}
		l := logic.NewImageLogic(r.Context(), svcCtx)
		resp, err := l.List(req.NodeId)
		if err != nil {
			fail(w, err)
			return
		}
		ok(w, resp)
	}
}

func imagePullHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, okReq := parse[types.PullImageReq](w, r)
		if !okReq {
			return
		}
		l := logic.NewImageLogic(r.Context(), svcCtx)
		resp, err := l.Pull(req.NodeId, req.Ref)
		if err != nil {
			fail(w, err)
			return
		}
		ok(w, resp)
	}
}

func imageRemoveHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, okReq := parse[types.RemoveImageReq](w, r)
		if !okReq {
			return
		}
		l := logic.NewImageLogic(r.Context(), svcCtx)
		if err := l.Remove(req.NodeId, req.Ref, req.Force); err != nil {
			fail(w, err)
			return
		}
		ok(w, nil)
	}
}

// ---------- dashboard ----------

func dashboardHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := logic.NewDashboardLogic(r.Context(), svcCtx)
		resp, err := l.Stats()
		if err != nil {
			fail(w, err)
			return
		}
		ok(w, resp)
	}
}
