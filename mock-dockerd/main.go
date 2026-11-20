// mock-dockerd —— Docker Engine API 模拟器
//
// 启动后监听 :2375(默认),实现 Docker Engine API v1.4x 的常用端点,
// docker SDK / docker CLI(-H tcp://host:2375)可无缝访问。
// 用于在拿到真实节点前模拟 Docker 对接联调。
//
// 用法:
//
//	mock-dockerd [-addr :2375] [-data state.json]
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	addr := flag.String("addr", ":2375", "listen address")
	data := flag.String("data", "", "path to persist state as JSON (empty = memory only)")
	flag.Parse()

	s := newStore(*data)
	if !s.restore() {
		s.mu.Lock()
		s.seed()
		s.mu.Unlock()
	}
	stop := make(chan struct{})
	s.startLogPumper(stop)

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		close(stop)
		s.mu.Lock()
		s.persistLocked()
		s.mu.Unlock()
		os.Exit(0)
	}()

	fmt.Printf("[mock-dockerd] listening on %s (Docker Engine API v%s simulator)\n", *addr, mockAPIVersion)
	log.Fatal(http.ListenAndServe(*addr, newRouter(s)))
}
