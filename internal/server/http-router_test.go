package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	cfg "github.com/edhollmon/reverse-proxy/internal/config"
)

var noTransport = cfg.HTTPTransportConfig{}

func TestHttpRouter_RouteMatch(t *testing.T) {
	var hit bool
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	router := &HttpRouter{}
	router.Add("/api", NewHttpLoadBalancer([]string{backend.URL}, noTransport))

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if !hit {
		t.Error("expected backend to be hit")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestHttpRouter_NoMatch(t *testing.T) {
	router := &HttpRouter{}

	req := httptest.NewRequest(http.MethodGet, "/unknown", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusBadGateway)
	}
}

func TestHttpLoadBalancer_RoundRobin(t *testing.T) {
	hits := make([]int, 2)
	backends := make([]*httptest.Server, 2)
	for i := range backends {
		idx := i
		backends[idx] = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hits[idx]++
			w.WriteHeader(http.StatusOK)
		}))
		defer backends[idx].Close()
	}

	lb := NewHttpLoadBalancer([]string{backends[0].URL, backends[1].URL}, noTransport)

	for i := 0; i < 4; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()
		lb.ServeHTTP(rr, req)
	}

	// counter starts at 0; Add(1) before each pick:
	// req 1: 1%2=1, req 2: 2%2=0, req 3: 3%2=1, req 4: 4%2=0
	if hits[0] != 2 || hits[1] != 2 {
		t.Errorf("expected 2 hits each, got backend[0]=%d backend[1]=%d", hits[0], hits[1])
	}
}

func TestNewHttpLoadBalancer_SkipsInvalidURL(t *testing.T) {
	// "%%invalid" has an invalid percent-encoding, url.Parse returns an error
	lb := NewHttpLoadBalancer([]string{"%%invalid", "localhost:9999"}, noTransport)
	if len(lb.backends) != 1 {
		t.Errorf("expected 1 valid backend, got %d", len(lb.backends))
	}
}
