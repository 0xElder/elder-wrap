package config

import (
	"fmt"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

const DefaultElderWrapPort = "8546"

type Config struct {
	ElderGrpcEndpoint string                   `yaml:"elder_grpc_endpoint"`
	ElderWrapPort     string                   `yaml:"elder_wrap_port"`
	RollAppConfigs    map[string]RollAppConfig `yaml:"rollup_rpcs"`
	KeyStoreDir       string                   `yaml:"key_store_dir"`
}

func (c *Config) validate() error {
	if c.ElderGrpcEndpoint == "" {
		return fmt.Errorf("elder_grpc_endpoint is required")
	}
	if c.ElderWrapPort == "" {
		c.ElderWrapPort = DefaultElderWrapPort
	}
	if len(c.RollAppConfigs) == 0 {
		return fmt.Errorf("rollup_rpcs is required")
	}
	for _, r := range c.RollAppConfigs {
		if err := r.validate(); err != nil {
			return err
		}
	}
	if c.KeyStoreDir == "" {
		return fmt.Errorf("key_store_dir is required")
	}
	return nil
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

type RollAppConfig struct {
	RPC                string `yaml:"rpc"`
	ElderRegistationId uint64 `yaml:"elder_registation_id"`
}

func (r *RollAppConfig) validate() error {
	if r.RPC == "" {
		return fmt.Errorf("rpc is required")
	}
	if r.ElderRegistationId <= 0 {
		return fmt.Errorf("elder_registation_id can't be negative or zero")
	}
	return nil
}

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
