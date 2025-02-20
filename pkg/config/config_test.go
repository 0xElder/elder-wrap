package config

import (
	"encoding/json"
	"os"
	"testing"
)

func TestConfig_validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				ElderGrpcEndpoint: "localhost:50051",
				ElderWrapPort:     "8546",
				RollAppConfigs: map[string]RollAppConfig{
					"rollup1": {
						RPC:                "http://localhost:8545",
						ElderRegistationId: 1,
					},
				},
				KeyStoreDir: "/tmp/keystore",
			},
			wantErr: false,
		},
		{
			name: "missing elder grpc endpoint",
			config: Config{
				ElderWrapPort: "8546",
				RollAppConfigs: map[string]RollAppConfig{
					"rollup1": {
						RPC:                "http://localhost:8545",
						ElderRegistationId: 1,
					},
				},
				KeyStoreDir: "/tmp/keystore",
			},
			wantErr: true,
		},
		{
			name: "missing rollapp configs",
			config: Config{
				ElderGrpcEndpoint: "localhost:50051",
				ElderWrapPort:     "8546",
				KeyStoreDir:       "/tmp/keystore",
			},
			wantErr: true,
		},
		{
			name: "invalid rollapp config - missing rpc",
			config: Config{
				ElderGrpcEndpoint: "localhost:50051",
				ElderWrapPort:     "8546",
				RollAppConfigs: map[string]RollAppConfig{
					"rollup1": {
						ElderRegistationId: 1,
					},
				},
				KeyStoreDir: "/tmp/keystore",
			},
			wantErr: true,
		},
		{
			name: "invalid rollapp config - invalid registration id",
			config: Config{
				ElderGrpcEndpoint: "localhost:50051",
				ElderWrapPort:     "8546",
				RollAppConfigs: map[string]RollAppConfig{
					"rollup1": {
						RPC:                "http://localhost:8545",
						ElderRegistationId: 0,
					},
				},
				KeyStoreDir: "/tmp/keystore",
			},
			wantErr: true,
		},
		{
			name: "missing keystore dir",
			config: Config{
				ElderGrpcEndpoint: "localhost:50051",
				ElderWrapPort:     "8546",
				RollAppConfigs: map[string]RollAppConfig{
					"rollup1": {
						RPC:                "http://localhost:8545",
						ElderRegistationId: 1,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "default port",
			config: Config{
				ElderGrpcEndpoint: "localhost:50051",
				RollAppConfigs: map[string]RollAppConfig{
					"rollup1": {
						RPC:                "http://localhost:8545",
						ElderRegistationId: 1,
					},
				},
				KeyStoreDir: "/tmp/keystore",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && tt.config.ElderWrapPort == "" {
				if tt.config.ElderWrapPort != DefaultElderWrapPort {
					t.Errorf("Config.validate() did not set default port, got %v, want %v",
						tt.config.ElderWrapPort, DefaultElderWrapPort)
				}
			}
		})
	}
}

func TestNewConfig(t *testing.T) {
	validConfig := Config{
		ElderGrpcEndpoint: "localhost:50051",
		ElderWrapPort:     "8546",
		RollAppConfigs: map[string]RollAppConfig{
			"rollup1": {
				RPC:                "http://localhost:8545",
				ElderRegistationId: 1,
			},
		},
		KeyStoreDir: "/tmp/keystore",
	}

	content, err := json.Marshal(validConfig)
	if err != nil {
		t.Fatal(err)
	}

	tmpfile, err := os.CreateTemp("", "config.*.json")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	originalConfigPath := "config.json"
	if err := os.Rename(originalConfigPath, originalConfigPath+".bak"); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	defer func() {
		os.Remove(originalConfigPath)
		if _, err := os.Stat(originalConfigPath + ".bak"); err == nil {
			os.Rename(originalConfigPath+".bak", originalConfigPath)
		}
	}()

	if err := os.Link(tmpfile.Name(), originalConfigPath); err != nil {
		t.Fatal(err)
	}

	config := NewConfig()
	if config == nil {
		t.Fatal("NewConfig() returned nil")
	}

	if config.ElderGrpcEndpoint != validConfig.ElderGrpcEndpoint {
		t.Errorf("NewConfig() ElderGrpcEndpoint = %v, want %v",
			config.ElderGrpcEndpoint, validConfig.ElderGrpcEndpoint)
	}

	if config.ElderWrapPort != validConfig.ElderWrapPort {
		t.Errorf("NewConfig() ElderWrapPort = %v, want %v",
			config.ElderWrapPort, validConfig.ElderWrapPort)
	}

	if len(config.RollAppConfigs) != len(validConfig.RollAppConfigs) {
		t.Errorf("NewConfig() RollAppConfigs count = %v, want %v",
			len(config.RollAppConfigs), len(validConfig.RollAppConfigs))
	}

	if config.KeyStoreDir != validConfig.KeyStoreDir {
		t.Errorf("NewConfig() KeyStoreDir = %v, want %v",
			config.KeyStoreDir, validConfig.KeyStoreDir)
	}
}
