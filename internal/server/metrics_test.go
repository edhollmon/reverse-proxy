package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestMetricsServer_Endpoint(t *testing.T) {
	srv := newMetricsServer("0")
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "text/plain") {
		t.Errorf("Content-Type = %q, want text/plain", ct)
	}
}

func TestMetricsServer_UnknownRoute(t *testing.T) {
	srv := newMetricsServer("0")
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rr.Code)
	}
}

func TestHTTPMetrics_RequestsRecorded(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	router := &HttpRouter{}
	router.Add("/metrics-test", NewHttpLoadBalancer([]string{backend.URL}))

	before := testutil.ToFloat64(httpRequestsTotal.WithLabelValues("/metrics-test", "200"))

	req := httptest.NewRequest(http.MethodGet, "/metrics-test/foo", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	after := testutil.ToFloat64(httpRequestsTotal.WithLabelValues("/metrics-test", "200"))
	if after-before != 1 {
		t.Errorf("httpRequestsTotal delta = %v, want 1", after-before)
	}
}

func TestHTTPMetrics_DurationRecorded(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer backend.Close()

	router := &HttpRouter{}
	router.Add("/metrics-dur", NewHttpLoadBalancer([]string{backend.URL}))

	req := httptest.NewRequest(http.MethodGet, "/metrics-dur/bar", nil)
	router.ServeHTTP(httptest.NewRecorder(), req)

	// Verify the histogram was observed by checking the /metrics output.
	srv := newMetricsServer("0")
	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rr, metricsReq)

	if !strings.Contains(rr.Body.String(), `reverse_proxy_http_request_duration_seconds_count{prefix="/metrics-dur"}`) {
		t.Error("expected duration histogram count in /metrics output")
	}
}
