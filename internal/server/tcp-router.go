package server

import (
	"errors"
	"io"
	"log/slog"
	"net"
	"os"
	"sync"
	"sync/atomic"
)

type TcpRouter struct {
	lbs []*TcpLoadBalancer
}

func (router *TcpRouter) Add(listenAddr string, backends []string) {
	router.lbs = append(router.lbs, NewTcpLoadBalancer(listenAddr, backends))
}

func (router *TcpRouter) Start() {
	for _, lb := range router.lbs {
		go lb.server.start()
	}
}

type TcpLoadBalancer struct {
	counter  atomic.Uint64
	backends []string
	server   *TcpServer
}

func NewTcpLoadBalancer(listenAddr string, backends []string) *TcpLoadBalancer {
	lb := &TcpLoadBalancer{backends: backends}
	lb.server = newTCPServer(listenAddr, lb)
	return lb
}

func (lb *TcpLoadBalancer) next() string {
	i := lb.counter.Add(1) % uint64(len(lb.backends))
	return lb.backends[i]
}

type tcpclient struct {
	cid     uint64
	conn    net.Conn
	backend net.Conn
}

func (c *tcpclient) proxy() {
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, _ = io.Copy(c.backend, c.conn)
		_ = c.backend.Close()
	}()
	go func() {
		defer wg.Done()
		_, _ = io.Copy(c.conn, c.backend)
		_ = c.conn.Close()
	}()
	wg.Wait()
}

type TcpServer struct {
	mu      sync.RWMutex
	listener net.Listener
	address string
	lb      *TcpLoadBalancer
	grWG    sync.WaitGroup
	clients map[uint64]*tcpclient
	nextcid atomic.Uint64
}

func newTCPServer(address string, lb *TcpLoadBalancer) *TcpServer {
	return &TcpServer{
		address: address,
		lb:      lb,
		clients: make(map[uint64]*tcpclient),
	}
}

func (s *TcpServer) start() {
	s.mu.Lock()
	l, err := net.Listen("tcp", s.address)
	if err != nil {
		s.mu.Unlock()
		slog.Error("TCP proxy failed to listen", "addr", s.address, "error", err)
		os.Exit(1)
	}
	s.listener = l
	s.mu.Unlock()

	s.grWG.Add(1)
	defer s.grWG.Done()

	for {
		conn, err := l.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				break
			}
			slog.Error("error accepting connection", "error", err)
			continue
		}
		s.grWG.Add(1)
		go s.handleConn(conn)
	}
}

func (s *TcpServer) handleConn(conn net.Conn) {
	defer s.grWG.Done()

	cid := s.nextcid.Add(1)
	slog.Info("client connected", "cid", cid, "remote_addr", conn.RemoteAddr())

	backendAddr := s.lb.next()
	backendConn, err := net.Dial("tcp", backendAddr)
	if err != nil {
		slog.Error("failed to connect to backend", "cid", cid, "backend", backendAddr, "error", err)
		_ = conn.Close()
		return
	}

	c := &tcpclient{cid: cid, conn: conn, backend: backendConn}

	s.mu.Lock()
	s.clients[cid] = c
	s.mu.Unlock()

	slog.Info("client proxying", "cid", cid, "backend", backendAddr)
	c.proxy()

	s.mu.Lock()
	delete(s.clients, cid)
	s.mu.Unlock()
	slog.Info("client disconnected", "cid", cid)
}