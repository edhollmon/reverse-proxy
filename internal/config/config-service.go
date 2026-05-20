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
	Prefix     string
	Port       string
	lbstrategy string
	Backends   []string
}

func (c *Connection) UnmarshalJSON(data []byte) error {
	var v struct {
		ConnType   string       `json:"type"`
		Prefix     string       `json:"prefix"`
		Port       string       `json:"port"`
		LBStrategy string       `json:"lbstrategy"`
		Hosts      []HostDetail `json:"hosts"`
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
	return nil
}

type TCPConnectionConfig struct {
	Connections []Connection
}

type HTTPConnectionConfig struct {
	Connections []Connection
}

type ConfigService struct {
	Tcp  TCPConnectionConfig
	Http HTTPConnectionConfig
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

	cs.Tcp = TCPConnectionConfig{Connections: cfg.Connections.TCP}
	cs.Http = HTTPConnectionConfig{Connections: cfg.Connections.HTTP}

	return nil
}
