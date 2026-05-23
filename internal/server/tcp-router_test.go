package server

import (
	"io"
	"net"
	"testing"
	"time"
)

func TestTcpLoadBalancer_Next_RoundRobin(t *testing.T) {
	lb := &TcpLoadBalancer{backends: []string{"a:1", "b:2", "c:3"}}

	// counter starts at 0; Add(1) returns new value before modulo
	want := []string{"b:2", "c:3", "a:1", "b:2"}
	for i, w := range want {
		if got := lb.next(); got != w {
			t.Errorf("call %d: next() = %q, want %q", i+1, got, w)
		}
	}
}

func TestTcpclient_Proxy(t *testing.T) {
	// net.Pipe gives two in-memory connected ends
	clientConn, proxyClient := net.Pipe()
	proxyBackend, backendConn := net.Pipe()
	defer func() { _ = clientConn.Close() }()
	defer func() { _ = backendConn.Close() }()

	c := &tcpclient{cid: 1, conn: proxyClient, backend: proxyBackend}

	done := make(chan struct{})
	go func() {
		c.proxy()
		close(done)
	}()

	// client → backend
	if _, err := clientConn.Write([]byte("ping")); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 4)
	if _, err := io.ReadFull(backendConn, buf); err != nil {
		t.Fatalf("backend read: %v", err)
	}
	if string(buf) != "ping" {
		t.Errorf("backend got %q, want %q", buf, "ping")
	}

	// backend → client
	if _, err := backendConn.Write([]byte("pong")); err != nil {
		t.Fatal(err)
	}
	buf2 := make([]byte, 4)
	if _, err := io.ReadFull(clientConn, buf2); err != nil {
		t.Fatalf("client read: %v", err)
	}
	if string(buf2) != "pong" {
		t.Errorf("client got %q, want %q", buf2, "pong")
	}

	_ = clientConn.Close()
	_ = backendConn.Close()
	<-done
}

func TestTcpServer_ProxiesData(t *testing.T) {
	// Start a simple echo backend
	backendLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = backendLn.Close() }()

	go func() {
		conn, err := backendLn.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		_, _ = io.Copy(conn, conn)
	}()

	lb := NewTcpLoadBalancer("127.0.0.1:0", []string{backendLn.Addr().String()})
	go lb.server.start()

	proxyAddr := waitForListener(t, lb.server)

	conn, err := net.DialTimeout("tcp", proxyAddr, 2*time.Second)
	if err != nil {
		t.Fatalf("dial proxy: %v", err)
	}
	defer func() { _ = conn.Close() }()

	if _, err := conn.Write([]byte("hello")); err != nil {
		t.Fatal(err)
	}

	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 5)
	if _, err := io.ReadFull(conn, buf); err != nil {
		t.Fatalf("read from proxy: %v", err)
	}
	if string(buf) != "hello" {
		t.Errorf("got %q, want %q", buf, "hello")
	}
}

func waitForListener(t *testing.T, s *TcpServer) string {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		s.mu.RLock()
		l := s.listener
		s.mu.RUnlock()
		if l != nil {
			return l.Addr().String()
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("TCP server did not start in time")
	return ""
}
