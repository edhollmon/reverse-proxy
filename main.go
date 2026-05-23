package main

import (
	"flag"
	"log/slog"
	"os"

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
	if err := rp.Start(); err != nil {
		slog.Error("failed to start server", "error", err)
		os.Exit(1)
	}

	slog.Info("server shutting down")
}
