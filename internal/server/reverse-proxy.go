package server

import (
	"log/slog"
	"net/http"
	"os"
	"strings"

	cfg "github.com/edhollmon/reverse-proxy/internal/config"
)

type ReverseProxy struct {
	configService *cfg.ConfigService
}

func NewReverseProxy() *ReverseProxy {
	return &ReverseProxy{}
}

func (rp *ReverseProxy) Start() error {
	if err := rp.loadConfig(); err != nil {
		return err
	}
	go rp.startTcpProxy()
	rp.startHttpProxy()
	return nil
}

func (rp *ReverseProxy) loadConfig() error {
	cs := cfg.NewConfigService()
	if err := cs.LoadDefaultConfig(); err != nil {
		return err
	}
	rp.configService = cs
	return nil
}

func (rp *ReverseProxy) startHttpProxy() {
	router := &HttpRouter{}
	for _, httpConfig := range rp.configService.Http.Connections {
		router.Add(httpConfig.Prefix, NewHttpLoadBalancer(httpConfig.Backends))
	}
	slog.Info("HTTP reverse proxy listening", "port", 8080)
	if err := http.ListenAndServe(":8080", router); err != nil {
		slog.Error("HTTP server stopped", "error", err)
		os.Exit(1)
	}
}

func (rp *ReverseProxy) startTcpProxy() {
	router := &TcpRouter{}
	for i, tcpConfig := range rp.configService.Tcp.Connections {
		slog.Info("TCP reverse proxy listening", "port", tcpConfig.Port, "backends", tcpConfig.Backends)
		for j, backend := range tcpConfig.Backends {
			slog.Info("TCP backend hint",
				"terminal", i*len(tcpConfig.Backends)+j+1,
				"backend_label", string(rune('A'+j)),
				"nc_port", backend[strings.LastIndex(backend, ":")+1:],
			)
		}
		router.Add(":"+tcpConfig.Port, tcpConfig.Backends)
	}
	router.Start()
}
