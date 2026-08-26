package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"tape-preservation-gate/internal/quality"
	"tape-preservation-gate/internal/store"
	webapp "tape-preservation-gate/internal/web"
	"tape-preservation-gate/internal/workflow"
	"time"
)

const defaultAddr = "127.0.0.1:19081"

func main() {
	if err := run(); err != nil {
		log.Printf("服务退出：%v", err)
		os.Exit(1)
	}
}

func run() error {
	addrFlag := flag.String("addr", defaultAddr, "监听地址，必须使用回环地址")
	dataDir := flag.String("data-dir", "data", "本地持久化目录")
	selfcheck := flag.Bool("selfcheck", false, "运行真实 HTTP 有界自检后退出")
	flag.Parse()
	addr := *addrFlag
	addrExplicit := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "addr" {
			addrExplicit = true
		}
	})
	if !addrExplicit {
		if port := os.Getenv("PORT"); port != "" {
			if _, err := strconv.Atoi(port); err != nil {
				return fmt.Errorf("PORT 必须为端口号")
			}
			addr = net.JoinHostPort("127.0.0.1", port)
		}
	}
	if err := validateAddr(addr); err != nil {
		return err
	}
	actualDir := *dataDir
	if *selfcheck {
		tmp, err := os.MkdirTemp("", "tape-preservation-selfcheck-")
		if err != nil {
			return err
		}
		defer os.RemoveAll(tmp)
		actualDir = tmp
	}
	repo, err := store.Open(actualDir)
	if err != nil {
		return fmt.Errorf("恢复持久化数据: %w", err)
	}
	defer repo.Close()
	service := workflow.New(repo, quality.NewEngine())
	handler := webapp.New(service).Handler()
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("监听 %s: %w", addr, err)
	}
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 20 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 32 << 10}
	if *selfcheck {
		return runSelfcheck(server, listener)
	}
	log.Printf("磁带数字化质量验收工作台监听 http://%s/app", addr)
	errCh := make(chan error, 1)
	go func() { errCh <- server.Serve(listener) }()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)
	select {
	case sig := <-sigCh:
		log.Printf("收到 %s，开始优雅关闭", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(ctx)
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func validateAddr(addr string) error {
	host, portText, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("-addr 格式无效: %w", err)
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("-addr 必须使用 127.0.0.1 或其他回环 IP")
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1024 || port > 65535 {
		return fmt.Errorf("监听端口必须是 1024 到 65535")
	}
	return nil
}
