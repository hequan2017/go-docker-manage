package main

import (
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strings"
)

// docker SDK 请求路径带版本前缀(如 /v1.47/containers/json),剥离后路由
var apiVersionPrefix = regexp.MustCompile(`^/v\d+\.\d+(/|$)`)

func stripVersionPrefix(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if apiVersionPrefix.MatchString(r.URL.Path) {
			r.URL.Path = apiVersionPrefix.ReplaceAllString(r.URL.Path, "/")
			if r.URL.Path == "" {
				r.URL.Path = "/"
			}
		}
		next.ServeHTTP(w, r)
	})
}

// writeJSON 标准响应
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError Docker 风格错误响应 {"message": "..."},SDK 依赖此格式映射 errdefs
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"message": msg})
}

// writeNoContent 204
func writeNoContent(w http.ResponseWriter) { w.WriteHeader(http.StatusNoContent) }

// notFoundContainer / notFoundImage 标准错误
func notFoundContainer(w http.ResponseWriter, ref string) {
	writeError(w, http.StatusNotFound, "No such container: "+ref)
}

func notFoundImage(w http.ResponseWriter, ref string) {
	writeError(w, http.StatusNotFound, "No such image: "+ref)
}

func newRouter(s *store) http.Handler {
	mux := http.NewServeMux()

	// ---- system ----
	mux.HandleFunc("/_ping", handlePing)
	mux.HandleFunc("/version", handleVersion)
	mux.HandleFunc("/info", handleInfo(s))

	// ---- containers ----
	mux.HandleFunc("/containers/json", handleContainerList(s))
	mux.HandleFunc("/containers/create", handleContainerCreate(s))
	mux.HandleFunc("/containers/", func(w http.ResponseWriter, r *http.Request) {
		// /containers/{id}            DELETE 删除
		// /containers/{id}/{action}   其余操作
		rest := strings.TrimPrefix(r.URL.Path, "/containers/")
		parts := strings.Split(rest, "/")
		if len(parts) == 1 && parts[0] != "" {
			if r.Method == http.MethodDelete {
				handleContainerRemove(s, w, r, parts[0])
				return
			}
			notFoundContainer(w, rest)
			return
		}
		if len(parts) < 2 {
			notFoundContainer(w, rest)
			return
		}
		id, action := parts[0], parts[1]
		switch action {
		case "json":
			handleContainerInspect(s, w, r, id)
		case "start", "stop", "restart", "kill", "pause", "unpause":
			handleContainerAction(s, w, r, id, action)
		case "stats":
			handleContainerStats(s, w, r, id)
		case "logs":
			handleContainerLogs(s, w, r, id)
		case "attach", "wait", "resize", "update", "rename", "top", "changes", "export", "archive":
			writeError(w, http.StatusNotImplemented, "mock-dockerd: "+action+" is not implemented")
		default:
			notFoundContainer(w, id)
		}
	})

	// ---- images ----
	mux.HandleFunc("/images/json", handleImageList(s))
	mux.HandleFunc("/images/create", handleImageCreate(s))
	mux.HandleFunc("/images/", func(w http.ResponseWriter, r *http.Request) {
		// DELETE /images/{name:.*}
		rest := strings.TrimPrefix(r.URL.Path, "/images/")
		if rest == "" || rest == "search" {
			writeError(w, http.StatusNotImplemented, "mock-dockerd: not implemented")
			return
		}
		handleImageRemove(s, w, r, rest)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[mock-dockerd] 404 %s %s", r.Method, r.URL.Path)
		writeError(w, http.StatusNotFound, "page not found")
	})

	return stripVersionPrefix(mux)
}
