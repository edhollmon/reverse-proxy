package server

import (
	"strings"
	"testing"

	cfg "github.com/edhollmon/reverse-proxy/internal/config"
)

func TestReverseProxy_Start_NoConnections(t *testing.T) {
	rp := NewReverseProxy(cfg.NewConfigService())
	err := rp.Start()
	if err == nil {
		t.Fatal("expected error for empty config, got nil")
	}
	if !strings.Contains(err.Error(), "no connections configured") {
		t.Errorf("error = %q, want it to contain 'no connections configured'", err.Error())
	}
}
