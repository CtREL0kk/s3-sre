package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"myS3/internal/app"
	"myS3/internal/config"
	"myS3/internal/logger"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "healthcheck":
			os.Exit(runHealthcheck())
		case "migrate":
			os.Exit(runAdmin("migrate", func(a *app.Application, ctx context.Context) error {
				return a.Migrate(ctx)
			}))
		}
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	l, err := logger.New(cfg.LogLevel, cfg.Env)
	if err != nil {
		log.Fatalf("init logger: %v", err)
	}
	defer func() { _ = l.Sync() }()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	application := app.NewApplication(cfg, l)
	if err := application.Init(ctx); err != nil {
		l.Errorw("failed to init application", "error", err)
		return
	}
	if err := application.Run(ctx); err != nil {
		l.Errorw("failed to run application", "error", err)
		return
	}
}

func runAdmin(name string, fn func(a *app.Application, ctx context.Context) error) int {
	cfg, err := config.Load()
	if err != nil {
		log.Printf("%s: load config: %v", name, err)
		return 1
	}
	l, err := logger.New(cfg.LogLevel, cfg.Env)
	if err != nil {
		log.Printf("%s: init logger: %v", name, err)
		return 1
	}
	defer func() { _ = l.Sync() }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	application := app.NewApplication(cfg, l)
	if err := fn(application, ctx); err != nil {
		l.Errorw("admin task failed", "task", name, "error", err)
		return 1
	}
	l.Infow("admin task completed", "task", name)
	return 0
}

func runHealthcheck() int {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	conn, err := net.DialTimeout("tcp", "127.0.0.1:"+port, 2*time.Second)
	if err != nil {
		return 1
	}
	_ = conn.Close()
	return 0
}
