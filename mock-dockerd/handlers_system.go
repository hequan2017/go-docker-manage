package main

import (
	"fmt"
	"net/http"
	"runtime"
	"time"
)

const (
	mockAPIVersion    = "1.47"
	mockServerVersion = "27.3.1"
	mockIDPrefix      = "GDMO" // daemon ID 前缀
)

// handlePing GET/HEAD /_ping —— SDK 版本协商依赖 Api-Version 响应头
func handlePing(w http.ResponseWriter, r *http.Request) {
	h := w.Header()
	h.Set("Api-Version", mockAPIVersion)
	h.Set("Docker-Experimental", "false")
	h.Set("Ostype", "linux")
	h.Set("Architecture", "x86_64")
	h.Set("Cache-Control", "no-cache, no-store, must-revalidate")
	h.Set("Pragma", "no-cache")
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodGet {
		_, _ = w.Write([]byte("OK"))
	}
}

// handleVersion GET /version
func handleVersion(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"Platform": map[string]any{
			"Name": "mock-dockerd (Docker Engine API Simulator)",
		},
		"Components": []map[string]any{
			{
				"Name":    "Engine",
				"Version": mockServerVersion,
				"Details": map[string]any{
					"ApiVersion":    mockAPIVersion,
					"MinAPIVersion": "1.24",
					"GitCommit":     "mock",
					"GoVersion":     runtime.Version(),
					"Os":            "linux",
					"Arch":          runtime.GOARCH,
					"KernelVersion": "6.1.0-mock",
					"BuildTime":     "2024-09-20T10:00:00Z",
				},
			},
		},
		"Version":       mockServerVersion,
		"ApiVersion":    mockAPIVersion,
		"MinAPIVersion": "1.24",
		"GitCommit":     "mock",
		"GoVersion":     runtime.Version(),
		"Os":            "linux",
		"Arch":          runtime.GOARCH,
		"KernelVersion": "6.1.0-mock",
		"BuildTime":     "2024-09-20T10:00:00Z",
	})
}

// handleInfo GET /info
func handleInfo(s *store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		total, running, paused, stopped := s.counts()
		writeJSON(w, http.StatusOK, map[string]any{
			"ID":                fmt.Sprintf("%s:%s", mockIDPrefix, mockServerVersion),
			"Containers":        total,
			"ContainersRunning": running,
			"ContainersPaused":  paused,
			"ContainersStopped": stopped,
			"Images":            len(s.listImages()),
			"Driver":            "overlay2",
			"MemoryLimit":       true,
			"CpuCfsPeriod":      true,
			"CpuCfsQuota":       true,
			"CPUShares":         true,
			"NCPU":              4,
			"MemTotal":          8 * 1024 * 1024 * 1024,
			"OperatingSystem":   "Alpine Linux v3.20 (mock)",
			"OSType":            "linux",
			"Architecture":      "x86_64",
			"ServerVersion":     mockServerVersion,
			"LoggingDriver":     "json-file",
			"CgroupDriver":      "cgroupfs",
			"KernelVersion":     "6.1.0-mock",
			"DriverStatus":      [][2]string{{"Backing Filesystem", "extfs"}},
			"SystemTime":        time.Now().Format(time.RFC3339Nano),
		})
	}
}
