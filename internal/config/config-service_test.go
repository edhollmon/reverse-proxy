package services

import (
	"os"
	"testing"
)

func TestLoadDefaultConfig(t *testing.T) {
	cs := NewConfigService()
	if err := cs.LoadDefaultConfig(); err != nil {
		t.Fatalf("LoadDefaultConfig() error: %v", err)
	}

	if len(cs.Tcp.Connections) == 0 {
		t.Fatal("expected TCP connections from default config")
	}
	if len(cs.Http.Connections) == 0 {
		t.Fatal("expected HTTP connections from default config")
	}

	tcp := cs.Tcp.Connections[0]
	if tcp.Port != "9090" {
		t.Errorf("TCP port = %q, want %q", tcp.Port, "9090")
	}
	if len(tcp.Backends) != 2 {
		t.Fatalf("TCP backends count = %d, want 2", len(tcp.Backends))
	}
	if tcp.Backends[0] != "localhost:9091" {
		t.Errorf("TCP backend[0] = %q, want %q", tcp.Backends[0], "localhost:9091")
	}
	if tcp.Backends[1] != "localhost:9092" {
		t.Errorf("TCP backend[1] = %q, want %q", tcp.Backends[1], "localhost:9092")
	}

	http := cs.Http.Connections[0]
	if http.Prefix != "/api" {
		t.Errorf("HTTP prefix = %q, want %q", http.Prefix, "/api")
	}
	if len(http.Backends) != 2 {
		t.Errorf("HTTP backends count = %d, want 2", len(http.Backends))
	}
}

func TestLoadConfig_Valid(t *testing.T) {
	const data = `{"connections":{"tcp":[{"type":"tcp","port":"7070","lbstrategy":"round-robin","hosts":[{"host":"127.0.0.1","port":"7071"}]}],"http":[]}}`

	f, err := os.CreateTemp("", "config-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.WriteString(data)
	f.Close()

	cs := NewConfigService()
	if err := cs.LoadConfig(f.Name()); err != nil {
		t.Fatalf("LoadConfig() error: %v", err)
	}
	if len(cs.Tcp.Connections) != 1 {
		t.Fatalf("got %d TCP connections, want 1", len(cs.Tcp.Connections))
	}
	if got := cs.Tcp.Connections[0].Backends[0]; got != "127.0.0.1:7071" {
		t.Errorf("backend = %q, want %q", got, "127.0.0.1:7071")
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	cs := NewConfigService()
	if err := cs.LoadConfig("/nonexistent/path/config.json"); err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

func TestLoadConfig_InvalidJSON(t *testing.T) {
	f, err := os.CreateTemp("", "config-*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	f.WriteString("not-valid-json{")
	f.Close()

	cs := NewConfigService()
	if err := cs.LoadConfig(f.Name()); err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}