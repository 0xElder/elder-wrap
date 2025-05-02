package config

import (
	"fmt"
	"log"
	"log/slog"
	"os"

	"gopkg.in/yaml.v3"
)

func NewConfig() *Config {
	file, err := os.ReadFile("config.yaml")
	if err != nil {
		log.Fatal(err)
	}

	var c Config
	err = yaml.Unmarshal(file, &c)
	if err != nil {
		log.Fatal(err)
	}

	err = c.validate()
	if err != nil {
		log.Fatal(err)
	}

	return &c
}

func (c *Config) GetRollAppConfig(name string) (*RollAppConfig, error) {
	r, ok := c.RollAppConfigs[name]
	if !ok {
		return nil, fmt.Errorf("rollApp %s not found", name)
	}
	return &r, nil
}

func (c *Config) ListRollApps() []string {
	var result []string
	for k := range c.RollAppConfigs {
		result = append(result, k)
	}
	return result
}

func (c *Config) GetSlogLevel() slog.Level {
	switch c.LogLevel {
	case "info":
		return slog.LevelInfo
	case "debug":
		return slog.LevelDebug
	case "error":
		return slog.LevelError
	case "warn":
		return slog.LevelWarn
	default:
		return slog.LevelInfo
	}
}

func (r *RollAppConfig) validate() error {
	if r.RPC == "" {
		return fmt.Errorf("rpc is required")
	}
	if r.ElderRegistrationId <= 0 {
		return fmt.Errorf("elder_registration_id can't be negative or zero")
	}
	return nil
}
