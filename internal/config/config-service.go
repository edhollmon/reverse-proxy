package services

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"
)

//go:embed default.config.json
var defaultConfig []byte

type HostDetail struct {
	host string
	port string
}

func (h *HostDetail) UnmarshalJSON(data []byte) error {
	var v struct {
		Host string `json:"host"`
		Port string `json:"port"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	h.host = v.Host
	h.port = v.Port
	return nil
}

type TCPTransportConfig struct {
	DialTimeout time.Duration
	KeepAlive   time.Duration
}

type HTTPTransportConfig struct {
	DialTimeout           time.Duration
	ResponseHeaderTimeout time.Duration
	IdleConnTimeout       time.Duration
	MaxIdleConns          int
	MaxIdleConnsPerHost   int
	MaxConnsPerHost       int
}

type Connection struct {
	connType     string
	Prefix       string
	Port         string
	lbstrategy   string
	Backends     []string
	TCPTransport TCPTransportConfig
	HTTPTransport HTTPTransportConfig
}

func (c *Connection) UnmarshalJSON(data []byte) error {
	var v struct {
		ConnType   string       `json:"type"`
		Prefix     string       `json:"prefix"`
		Port       string       `json:"port"`
		LBStrategy string       `json:"lbstrategy"`
		Hosts      []HostDetail `json:"hosts"`
		Transport  struct {
			DialTimeout           string `json:"dialTimeout"`
			KeepAlive             string `json:"keepAlive"`
			ResponseHeaderTimeout string `json:"responseHeaderTimeout"`
			IdleConnTimeout       string `json:"idleConnTimeout"`
			MaxIdleConns          int    `json:"maxIdleConns"`
			MaxIdleConnsPerHost   int    `json:"maxIdleConnsPerHost"`
			MaxConnsPerHost       int    `json:"maxConnsPerHost"`
		} `json:"transport"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	c.connType = v.ConnType
	c.Prefix = v.Prefix
	c.Port = v.Port
	c.lbstrategy = v.LBStrategy
	c.Backends = make([]string, len(v.Hosts))
	for i, h := range v.Hosts {
		c.Backends[i] = h.host + ":" + h.port
	}
	t := v.Transport
	c.HTTPTransport.MaxIdleConns = t.MaxIdleConns
	c.HTTPTransport.MaxIdleConnsPerHost = t.MaxIdleConnsPerHost
	c.HTTPTransport.MaxConnsPerHost = t.MaxConnsPerHost
	for _, pair := range []struct {
		s   string
		dst *time.Duration
		key string
	}{
		{t.DialTimeout, &c.HTTPTransport.DialTimeout, "dialTimeout"},
		{t.DialTimeout, &c.TCPTransport.DialTimeout, "dialTimeout"},
		{t.ResponseHeaderTimeout, &c.HTTPTransport.ResponseHeaderTimeout, "responseHeaderTimeout"},
		{t.IdleConnTimeout, &c.HTTPTransport.IdleConnTimeout, "idleConnTimeout"},
		{t.KeepAlive, &c.TCPTransport.KeepAlive, "keepAlive"},
	} {
		if pair.s != "" {
			d, err := time.ParseDuration(pair.s)
			if err != nil {
				return fmt.Errorf("invalid %s %q: %w", pair.key, pair.s, err)
			}
			*pair.dst = d
		}
	}
	return nil
}

type TCPConnectionConfig struct {
	Connections []Connection
}

type HTTPConnectionConfig struct {
	Connections []Connection
}

type MetricsConfig struct {
	Port    string
	Enabled bool
}

type ConfigService struct {
	Tcp     TCPConnectionConfig
	Http    HTTPConnectionConfig
	Metrics MetricsConfig
	// Web Sockets, gRPC, ...
}

func applyTCPTransportDefaults(t *TCPTransportConfig) {
	if t.DialTimeout == 0 {
		t.DialTimeout = 5 * time.Second
	}
	if t.KeepAlive == 0 {
		t.KeepAlive = 30 * time.Second
	}
}

func applyHTTPTransportDefaults(t *HTTPTransportConfig) {
	if t.DialTimeout == 0 {
		t.DialTimeout = 5 * time.Second
	}
	if t.ResponseHeaderTimeout == 0 {
		t.ResponseHeaderTimeout = 30 * time.Second
	}
	if t.IdleConnTimeout == 0 {
		t.IdleConnTimeout = 90 * time.Second
	}
	if t.MaxIdleConns == 0 {
		t.MaxIdleConns = 100
	}
	if t.MaxIdleConnsPerHost == 0 {
		t.MaxIdleConnsPerHost = 20
	}
	if t.MaxConnsPerHost == 0 {
		t.MaxConnsPerHost = 200
	}
}

var validLBStrategies = map[string]bool{
	"round-robin":       true,
	"least-connections": true,
}

func (c *Connection) validate(label string) error {
	if !validLBStrategies[c.lbstrategy] {
		return fmt.Errorf("%s: unknown lbstrategy %q, must be one of: round-robin, least-connections", label, c.lbstrategy)
	}
	if c.connType == "tcp" && c.Port == "" {
		return fmt.Errorf("%s: missing port", label)
	}
	if len(c.Backends) == 0 {
		return fmt.Errorf("%s: must have at least one host", label)
	}
	for i, b := range c.Backends {
		host, port, err := net.SplitHostPort(b)
		if err != nil || host == "" || port == "" {
			return fmt.Errorf("%s host[%d]: invalid host:port %q", label, i, b)
		}
	}
	return nil
}

func NewConfigService() *ConfigService {
	return &ConfigService{}
}

func (cs *ConfigService) LoadDefaultConfig() error {
	return cs.parseConfig(defaultConfig)
}

func (cs *ConfigService) LoadConfig(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("config file not found: %s", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	return cs.parseConfig(data)
}

func (cs *ConfigService) parseConfig(data []byte) error {
	var cfg struct {
		Connections struct {
			TCP  []Connection `json:"tcp"`
			HTTP []Connection `json:"http"`
		} `json:"connections"`
		Metrics struct {
			Port    string `json:"port"`
			Enabled *bool  `json:"enabled"`
		} `json:"metrics"`
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	if cfg.Metrics.Port == "" {
		cfg.Metrics.Port = "9090"
	}
	metricsEnabled := cfg.Metrics.Enabled == nil || *cfg.Metrics.Enabled

	for i := range cfg.Connections.TCP {
		applyTCPTransportDefaults(&cfg.Connections.TCP[i].TCPTransport)
	}
	for i := range cfg.Connections.HTTP {
		applyHTTPTransportDefaults(&cfg.Connections.HTTP[i].HTTPTransport)
	}

	cs.Tcp = TCPConnectionConfig{Connections: cfg.Connections.TCP}
	cs.Http = HTTPConnectionConfig{Connections: cfg.Connections.HTTP}
	cs.Metrics = MetricsConfig{Port: cfg.Metrics.Port, Enabled: metricsEnabled}

	for i, c := range cs.Tcp.Connections {
		if err := c.validate(fmt.Sprintf("tcp[%d]", i)); err != nil {
			return err
		}
	}
	for i, c := range cs.Http.Connections {
		if err := c.validate(fmt.Sprintf("http[%d]", i)); err != nil {
			return err
		}
	}

	return nil
}
