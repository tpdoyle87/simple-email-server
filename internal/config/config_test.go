package config

import (
	"testing"
	"time"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				Server: ServerConfig{
					Hostname: "mail.example.com",
				},
				API: APIConfig{
					AuthToken: "test-token",
				},
			},
			wantErr: false,
		},
		{
			name: "missing hostname",
			config: &Config{
				API: APIConfig{
					AuthToken: "test-token",
				},
			},
			wantErr: true,
		},
		{
			name: "missing auth token",
			config: &Config{
				Server: ServerConfig{
					Hostname: "mail.example.com",
				},
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			
			// Check defaults are set
			if !tt.wantErr && err == nil {
				if tt.config.Server.ListenAddress == "" {
					t.Error("Server.ListenAddress should have default value")
				}
				if tt.config.API.ListenAddress == "" {
					t.Error("API.ListenAddress should have default value")
				}
				if tt.config.Queue.MaxRetry == 0 {
					t.Error("Queue.MaxRetry should have default value")
				}
				if tt.config.Delivery.Workers == 0 {
					t.Error("Delivery.Workers should have default value")
				}
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	
	if cfg.Server.ListenAddress != "0.0.0.0:587" {
		t.Errorf("Expected server listen address '0.0.0.0:587', got '%s'", cfg.Server.ListenAddress)
	}
	
	if cfg.API.ListenAddress != "127.0.0.1:8080" {
		t.Errorf("Expected API listen address '127.0.0.1:8080', got '%s'", cfg.API.ListenAddress)
	}
	
	if cfg.Queue.MaxRetry != 5 {
		t.Errorf("Expected max retry 5, got %d", cfg.Queue.MaxRetry)
	}
	
	if cfg.Queue.RetryDelay != 5*time.Minute {
		t.Errorf("Expected retry delay 5m, got %v", cfg.Queue.RetryDelay)
	}
	
	if cfg.Delivery.Workers != 20 {
		t.Errorf("Expected 20 workers, got %d", cfg.Delivery.Workers)
	}
	
	if cfg.Limits.MaxMessageSize != 25*1024*1024 {
		t.Errorf("Expected max message size 25MB, got %d", cfg.Limits.MaxMessageSize)
	}
}