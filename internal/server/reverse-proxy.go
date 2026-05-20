package server

import (
	"fmt"
	"log"
	"net/http"
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
		log.Fatal(err)
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
	fmt.Println("HTTP Reverse Proxy listening on 8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}

func (rp *ReverseProxy) startTcpProxy() {
	router := &TcpRouter{}
	for i, tcpConfig := range rp.configService.Tcp.Connections {
		fmt.Printf("TCP Reverse Proxy listening on localhost:%s -> backends: %v\n", tcpConfig.Port, tcpConfig.Backends)
		for j, backend := range tcpConfig.Backends {
			fmt.Printf("  # terminal %d — backend %c\n  nc -l %s\n", i*len(tcpConfig.Backends)+j+1, rune('A'+j), backend[strings.LastIndex(backend, ":")+1:])
		}
		router.Add(":"+tcpConfig.Port, tcpConfig.Backends)
	}
	router.Start()
}
