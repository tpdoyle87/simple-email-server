package config

import (
	"fmt"
	"time"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	API      APIConfig      `yaml:"api"`
	Queue    QueueConfig    `yaml:"queue"`
	Delivery DeliveryConfig `yaml:"delivery"`
	Limits   LimitsConfig   `yaml:"limits"`
	Logging  LoggingConfig  `yaml:"logging"`
}

type ServerConfig struct {
	Hostname      string     `yaml:"hostname"`
	ListenAddress string     `yaml:"listen_address"`
	TLS           TLSConfig  `yaml:"tls"`
}

type TLSConfig struct {
	Enabled  bool   `yaml:"enabled"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
	AutoTLS  bool   `yaml:"auto_tls"`
}

type APIConfig struct {
	ListenAddress string `yaml:"listen_address"`
	AuthToken     string `yaml:"auth_token"`
	TLS           TLSConfig `yaml:"tls"`
}

type QueueConfig struct {
	StoragePath   string        `yaml:"storage_path"`
	MaxSize       int           `yaml:"max_queue_size"`
	MaxRetry      int           `yaml:"max_retry"`
	RetryDelay    time.Duration `yaml:"retry_delay"`
	BatchSize     int           `yaml:"batch_size"`
}

type DeliveryConfig struct {
	Workers            int           `yaml:"workers"`
	DNSCacheTTL        time.Duration `yaml:"dns_cache_ttl"`
	ConnectionTimeout  time.Duration `yaml:"connection_timeout"`
	ConnectionPoolSize int           `yaml:"connection_pool_size"`
}

type LimitsConfig struct {
	MaxRecipients   int    `yaml:"max_recipients"`
	MaxMessageSize  int64  `yaml:"max_message_size"`
	RateLimit       string `yaml:"rate_limit"`
}

type LoggingConfig struct {
	Level string `yaml:"level"`
	File  string `yaml:"file"`
}

func (c *Config) Validate() error {
	if c.Server.Hostname == "" {
		return fmt.Errorf("server.hostname is required")
	}
	
	if c.Server.ListenAddress == "" {
		c.Server.ListenAddress = "0.0.0.0:587"
	}
	
	if c.API.ListenAddress == "" {
		c.API.ListenAddress = "127.0.0.1:8080"
	}
	
	if c.API.AuthToken == "" {
		return fmt.Errorf("api.auth_token is required")
	}
	
	if c.Queue.MaxRetry == 0 {
		c.Queue.MaxRetry = 5
	}
	
	if c.Queue.RetryDelay == 0 {
		c.Queue.RetryDelay = 5 * time.Minute
	}
	
	if c.Queue.BatchSize == 0 {
		c.Queue.BatchSize = 100
	}
	
	if c.Queue.MaxSize == 0 {
		c.Queue.MaxSize = 10000
	}
	
	if c.Delivery.Workers == 0 {
		c.Delivery.Workers = 20
	}
	
	if c.Delivery.DNSCacheTTL == 0 {
		c.Delivery.DNSCacheTTL = 5 * time.Minute
	}
	
	if c.Delivery.ConnectionTimeout == 0 {
		c.Delivery.ConnectionTimeout = 30 * time.Second
	}
	
	if c.Delivery.ConnectionPoolSize == 0 {
		c.Delivery.ConnectionPoolSize = 100
	}
	
	if c.Limits.MaxRecipients == 0 {
		c.Limits.MaxRecipients = 100
	}
	
	if c.Limits.MaxMessageSize == 0 {
		c.Limits.MaxMessageSize = 25 * 1024 * 1024 // 25MB
	}
	
	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	
	return nil
}

func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			ListenAddress: "0.0.0.0:587",
		},
		API: APIConfig{
			ListenAddress: "127.0.0.1:8080",
		},
		Queue: QueueConfig{
			MaxSize:    10000,
			MaxRetry:   5,
			RetryDelay: 5 * time.Minute,
			BatchSize:  100,
		},
		Delivery: DeliveryConfig{
			Workers:            20,
			DNSCacheTTL:        5 * time.Minute,
			ConnectionTimeout:  30 * time.Second,
			ConnectionPoolSize: 100,
		},
		Limits: LimitsConfig{
			MaxRecipients:  100,
			MaxMessageSize: 25 * 1024 * 1024,
		},
		Logging: LoggingConfig{
			Level: "info",
		},
	}
}