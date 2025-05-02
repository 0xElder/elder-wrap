package config

import (
	"fmt"
)

const DefaultElderWrapPort = "8546"

type Config struct {
	ElderGrpcEndpoint string                   `yaml:"elder_grpc_endpoint"`
	ElderWrapPort     string                   `yaml:"elder_wrap_port"`
	RollAppConfigs    map[string]RollAppConfig `yaml:"rollup_rpcs"`
	KeyStoreDir       string                   `yaml:"key_store_dir"`
	LogLevel          string                   `yaml:"log_level"`
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

type RollAppConfig struct {
	RPC                 string `yaml:"rpc"`
	ElderRegistrationId uint64 `yaml:"elder_registration_id"`
}
