package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

// ---------- GET /images/json ----------

func handleImageList(s *store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		images := s.listImages()
		out := make([]map[string]any, 0, len(images))
		for _, img := range images {
			tags := img.RepoTags
			if tags == nil {
				tags = []string{"<none>:<none>"}
			}
			out = append(out, map[string]any{
				"RepoTags":   tags,
				"Id":         img.ID,
				"Created":    img.Created,
				"Size":       img.Size,
				"Containers": -1,
			})
		}
		writeJSON(w, http.StatusOK, out)
	}
}

// ---------- POST /images/create?fromImage=nginx&tag=alpine ----------
// 模拟 registry 拉取:输出 JSONMessage 进度流,最后把镜像加入本地列表。
// 任何合法 ref 都能"拉取成功"(与真实 registry 行为对齐)。

func handleImageCreate(s *store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		fromImage := q.Get("fromImage")
		tag := q.Get("tag")
		if fromImage == "" && q.Get("fromSrc") == "" {
			writeError(w, http.StatusBadRequest, "fromImage or fromSrc required")
			return
		}
		ref := fromImage
		if tag != "" && !strings.Contains(fromImage, ":") {
			ref = fromImage + ":" + tag
		}
		if !strings.Contains(ref, ":") {
			ref += ":latest"
		}

		existed := s.findImage(ref)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		enc := json.NewEncoder(w)

		emit := func(v any) bool {
			if err := enc.Encode(v); err != nil {
				return false
			}
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			return true
		}

		if existed != nil {
			// 已存在:与真实 daemon 一致返回 up to date
			_ = emit(map[string]any{"status": "Status: Image is up to date for " + ref})
			return
		}

		repo := ref
		if i := strings.LastIndex(ref, ":"); i > 0 {
			repo = ref[:i]
		}
		if !emit(map[string]any{"status": "Pulling from library/" + repo, "id": tag}) {
			return
		}
		layers := 3 + rand.New(rand.NewSource(time.Now().UnixNano())).Intn(3)
		for i := 0; i < layers; i++ {
			layerID := newID()[:12]
			total := 1<<20 + rand.Int63n(30<<20)
			_ = emit(map[string]any{"status": "Pulling fs layer", "progressDetail": map[string]int{}, "id": layerID})
			_ = emit(map[string]any{
				"status":        "Downloading",
				"progressDetail": map[string]int64{"current": total, "total": total},
				"progress":      fmt.Sprintf("[==================================================>]  %dB", total),
				"id":            layerID,
			})
			time.Sleep(120 * time.Millisecond) // 让前端能看到"进度感"
			_ = emit(map[string]any{"status": "Verifying Checksum", "progressDetail": map[string]int{}, "id": layerID})
			_ = emit(map[string]any{"status": "Download complete", "progressDetail": map[string]int{}, "id": layerID})
			_ = emit(map[string]any{"status": "Extracting", "progressDetail": map[string]int{}, "id": layerID})
			_ = emit(map[string]any{"status": "Pull complete", "progressDetail": map[string]int{}, "id": layerID})
		}
		_ = emit(map[string]any{"status": "Digest: sha256:" + newID()})
		_ = emit(map[string]any{"status": "Status: Downloaded newer image for " + ref})

		s.upsertImage(&mImage{
			ID:       "sha256:" + newID(),
			RepoTags: []string{ref},
			Created:  time.Now().Unix(),
			Size:     (4 + rand.Int63n(500)) * 1024 * 1024,
		})
	}
}

// ---------- DELETE /images/{name:.*}?force=1 ----------

func handleImageRemove(s *store, w http.ResponseWriter, r *http.Request, ref string) {
	img := s.findImage(ref)
	if img == nil {
		notFoundImage(w, ref)
		return
	}
	force := r.URL.Query().Get("force") == "1"
	if !force && s.imageInUse(img) {
		writeError(w, http.StatusConflict,
			fmt.Sprintf("conflict: unable to delete %s (cannot be forced) - image is being used by running container", ref))
		return
	}
	removed := s.removeImage(img.ID)
	if removed == nil {
		notFoundImage(w, ref)
		return
	}
	out := []map[string]string{}
	for _, t := range removed.RepoTags {
		out = append(out, map[string]string{"Untagged": t})
	}
	out = append(out, map[string]string{"Deleted": removed.ID})
	writeJSON(w, http.StatusOK, out)
}
