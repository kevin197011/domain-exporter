package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config main configuration structure
type Config struct {
	Server  ServerConfig  `yaml:"server"`
	Checker CheckerConfig `yaml:"checker"`
	Domains []string      `yaml:"domains"`
}

// ServerConfig server configuration
type ServerConfig struct {
	Port        int    `yaml:"port"`
	MetricsPath string `yaml:"metrics_path"`
}

// CheckerConfig checker configuration
type CheckerConfig struct {
	CheckInterval int `yaml:"check_interval"`
	Concurrency   int `yaml:"concurrency"`
	Timeout       int `yaml:"timeout"`
}



// LoadConfig loads configuration file
func LoadConfig(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set default values
	if config.Server.Port == 0 {
		config.Server.Port = 8080
	}
	if config.Server.MetricsPath == "" {
		config.Server.MetricsPath = "/metrics"
	}
	if config.Checker.CheckInterval == 0 {
		config.Checker.CheckInterval = 3600
	}
	if config.Checker.Concurrency == 0 {
		config.Checker.Concurrency = 10
	}
	if config.Checker.Timeout == 0 {
		config.Checker.Timeout = 30
	}

	return &config, nil
}

// GetCheckInterval gets check interval duration
func (c *CheckerConfig) GetCheckInterval() time.Duration {
	return time.Duration(c.CheckInterval) * time.Second
}

// GetTimeout gets timeout duration
func (c *CheckerConfig) GetTimeout() time.Duration {
	return time.Duration(c.Timeout) * time.Second
}