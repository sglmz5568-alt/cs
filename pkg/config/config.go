package config

import (
	"os"
	"sync"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Security SecurityConfig `yaml:"security"`
	Logging  LoggingConfig  `yaml:"logging"`
	Rules    RulesConfig    `yaml:"rules"`
}

type ServerConfig struct {
	ProxyPort int    `yaml:"proxy_port"`
	WebPort   int    `yaml:"web_port"`
	BindIP    string `yaml:"bind_ip"`
}

type SecurityConfig struct {
	Enabled    bool     `yaml:"enabled"`
	APIToken   string   `yaml:"api_token"`
	AllowedIPs []string `yaml:"allowed_ips"`
}

type LoggingConfig struct {
	Level   string `yaml:"level"`
	Console bool   `yaml:"console"`
	File    string `yaml:"file"`
}

type RulesConfig struct {
	File string `yaml:"file"`
}

var (
	cfg  *Config
	once sync.Once
)

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	config := &Config{
		Server: ServerConfig{
			ProxyPort: 2021,
			WebPort:   2022,
			BindIP:    "0.0.0.0",
		},
		Security: SecurityConfig{
			Enabled:    true,
			APIToken:   "changeme",
			AllowedIPs: []string{"0.0.0.0/0"},
		},
		Logging: LoggingConfig{
			Level:   "info",
			Console: true,
		},
		Rules: RulesConfig{
			File: "rules.json",
		},
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, err
	}

	once.Do(func() {
		cfg = config
	})

	return config, nil
}

func Get() *Config {
	return cfg
}

func Default() *Config {
	return &Config{
		Server: ServerConfig{
			ProxyPort: 2021,
			WebPort:   2022,
			BindIP:    "0.0.0.0",
		},
		Security: SecurityConfig{
			Enabled:    false,
			APIToken:   "",
			AllowedIPs: []string{},
		},
		Logging: LoggingConfig{
			Level:   "info",
			Console: true,
		},
		Rules: RulesConfig{
			File: "rules.json",
		},
	}
}
