package services

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
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

type Connection struct {
	connType   string
	lbstrategy string
	hosts      []HostDetail
}

func (c *Connection) UnmarshalJSON(data []byte) error {
	var v struct {
		ConnType   string       `json:"type"`
		LBStrategy string       `json:"lbstrategy"`
		Hosts      []HostDetail `json:"hosts"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	c.connType = v.ConnType
	c.lbstrategy = v.LBStrategy
	c.hosts = v.Hosts
	return nil
}

type TCPConnectionConfig struct {
	connections []Connection
}

type HTTPConnectionConfig struct {
	connections []Connection
}

type ConfigService struct {
	tcp  TCPConnectionConfig
	http HTTPConnectionConfig
	// Web Sockets, gRPC, ...
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
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	cs.tcp = TCPConnectionConfig{connections: cfg.Connections.TCP}
	cs.http = HTTPConnectionConfig{connections: cfg.Connections.HTTP}

	return nil
}

func (cs *ConfigService) String() string {
	result := "TCP connections:\n"
	for _, conn := range cs.tcp.connections {
		result += fmt.Sprintf("  type=%s lbstrategy=%s\n", conn.connType, conn.lbstrategy)
		for _, h := range conn.hosts {
			result += fmt.Sprintf("    host=%s port=%s\n", h.host, h.port)
		}
	}
	result += "HTTP connections:\n"
	for _, conn := range cs.http.connections {
		result += fmt.Sprintf("  type=%s lbstrategy=%s\n", conn.connType, conn.lbstrategy)
		for _, h := range conn.hosts {
			result += fmt.Sprintf("    host=%s port=%s\n", h.host, h.port)
		}
	}
	return result
}
