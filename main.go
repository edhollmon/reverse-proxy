package main

import (
	"log/slog"
	"os"

	services "github.com/edhollmon/reverse-proxy/internal/config"
	"github.com/edhollmon/reverse-proxy/internal/server"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))

	cs := services.NewConfigService()
	if err := cs.LoadDefaultConfig(); err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}
	slog.Info("config loaded", "config", cs)

	rp := server.NewReverseProxy()
	if err := rp.Start(); err != nil {
		slog.Error("failed to start server", "error", err)
		os.Exit(1)
	}

	slog.Info("server shutting down")
}
