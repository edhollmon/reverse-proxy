package server

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	cfg "github.com/edhollmon/reverse-proxy/internal/config"
)

type Router struct {
	routes []Route
}

type Route struct {
	prefix string
	lb     *LoadBalancer
}

type LoadBalancer struct {
	proxy    *httputil.ReverseProxy
	counter  atomic.Uint64
	backends []*url.URL
}

func (lb *LoadBalancer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	lb.proxy.ServeHTTP(rw, req)
}

func (r *Router) Add(prefix string, lb *LoadBalancer) {
	r.routes = append(r.routes, Route{
		prefix: prefix,
		lb:     lb,
	})
}

func (router *Router) ServeHTTP(rw http.ResponseWriter, req *http.Request) {

	for _, route := range router.routes {
		if strings.HasPrefix(req.URL.Path, route.prefix) {
			route.lb.ServeHTTP(rw, req)
			return
		}
	}

	// TODO: Implement Default Route option
	http.Error(rw, "no route matched", http.StatusBadGateway)
}

func (route *Route) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	route.lb.ServeHTTP(rw, req)
}

func NewLoadBalancer(addrs []string) *LoadBalancer {
	lb := &LoadBalancer{}
	backends := make([]*url.URL, 0, len(addrs))
	for _, addr := range addrs {
		if !strings.Contains(addr, "://") {
			addr = "http://" + addr
		}
		u, err := url.Parse(addr)
		if err != nil {
			log.Printf("skipping invalid backend %q: %v", addr, err)
			continue
		}
		backends = append(backends, u)
	}
	lb.backends = backends

	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		MaxConnsPerHost:     200,
		DialContext: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).DialContext,
		ResponseHeaderTimeout: 30 * time.Second,
		IdleConnTimeout:       90 * time.Second,
	}
	lb.proxy = &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			target := lb.next()
			r.SetURL(target)
			r.SetXForwarded()
		},
		Transport: transport,
	}

	return lb
}

func (lb *LoadBalancer) next() *url.URL {
	i := lb.counter.Add(1) % uint64(len(lb.backends))
	return lb.backends[i]
}

type ReverseProxy struct {
	configService *cfg.ConfigService
}

func NewReverseProxy() *ReverseProxy {
	return &ReverseProxy{}
}

func (rp *ReverseProxy) Start() error {
	// Load config
	if err := rp.loadConfig(); err != nil {
		return err
	}

	// Spin up HTTP Host with the configuration
	router := &Router{}

	for _, httpConfig := range rp.configService.Http.Connections {
		router.Add(httpConfig.Prefix, NewLoadBalancer(httpConfig.Backends))
	}

	fmt.Println("Reverse Proxy listening on 8080")
	log.Fatal(http.ListenAndServe(":8080", router))

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
