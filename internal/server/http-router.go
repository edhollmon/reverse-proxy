package server

import (
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync/atomic"
	"time"
)

type HttpRouter struct {
	routes []HttpRoute
}

type HttpRoute struct {
	prefix string
	lb     *HttpLoadBalancer
}

type HttpLoadBalancer struct {
	proxy    *httputil.ReverseProxy
	counter  atomic.Uint64
	backends []*url.URL
}

func (lb *HttpLoadBalancer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	lb.proxy.ServeHTTP(rw, req)
}

func (r *HttpRouter) Add(prefix string, lb *HttpLoadBalancer) {
	r.routes = append(r.routes, HttpRoute{
		prefix: prefix,
		lb:     lb,
	})
}

func (router *HttpRouter) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	for _, route := range router.routes {
		if strings.HasPrefix(req.URL.Path, route.prefix) {
			route.lb.ServeHTTP(rw, req)
			return
		}
	}
	// TODO: Implement Default Route option
	http.Error(rw, "no route matched", http.StatusBadGateway)
}

func (route *HttpRoute) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	route.lb.ServeHTTP(rw, req)
}

func NewHttpLoadBalancer(addrs []string) *HttpLoadBalancer {
	lb := &HttpLoadBalancer{}
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

func (lb *HttpLoadBalancer) next() *url.URL {
	i := lb.counter.Add(1) % uint64(len(lb.backends))
	return lb.backends[i]
}