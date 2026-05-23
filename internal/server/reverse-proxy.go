package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	cfg "github.com/edhollmon/reverse-proxy/internal/config"
)

type ReverseProxy struct {
	configService *cfg.ConfigService
}

func NewReverseProxy(cs *cfg.ConfigService) *ReverseProxy {
	return &ReverseProxy{configService: cs}
}

func (rp *ReverseProxy) Start() error {
	hasTCP := len(rp.configService.Tcp.Connections) > 0
	hasHTTP := len(rp.configService.Http.Connections) > 0

	if !hasTCP && !hasHTTP {
		return fmt.Errorf("no connections configured")
	}

	if hasTCP && hasHTTP {
		go rp.startHttpProxy()
		rp.startTcpProxy()
		return nil
	}

	if hasHTTP {
		rp.startHttpProxy()
		return nil
	}

	rp.startTcpProxy()
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
