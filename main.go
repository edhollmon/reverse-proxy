package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	services "github.com/edhollmon/reverse-proxy/internal/config"
	"github.com/edhollmon/reverse-proxy/internal/server"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))

	configPath := flag.String("config", "", "path to config file (uses built-in default if omitted)")
	flag.Parse()

	cs := services.NewConfigService()

	if *configPath != "" {
		slog.Info("loading config", "path", *configPath)
		if err := cs.LoadConfig(*configPath); err != nil {
			slog.Error("failed to load config", "error", err)
			os.Exit(1)
		}
	} else {
		if err := cs.LoadDefaultConfig(); err != nil {
			slog.Error("failed to load default config", "error", err)
			os.Exit(1)
		}
	}

	slog.Info("config loaded", "config", cs)

	rp := server.NewReverseProxy(cs)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	serverErr := make(chan error, 1)
	go func() { serverErr <- rp.Start() }()

	select {
	case err := <-serverErr:
		if err != nil {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	case <-quit:
	}

	slog.Info("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := rp.Shutdown(ctx); err != nil {
		slog.Error("shutdown error", "error", err)
		os.Exit(1)
	}
	slog.Info("server stopped")
}
