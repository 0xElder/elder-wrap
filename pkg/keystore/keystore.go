package keystore

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/0xElder/elder/utils"
	"github.com/ethereum/go-ethereum/common"
)

var (
	ErrKeyNotFound = errors.New("key not found")
	ErrKeyExists   = errors.New("key already exists")
)

type Key struct {
	EvmAddress   common.Address            `json:"evmAddress"`
	ElderAddress string                    `json:"elderAddress"`
	PrivateKey   utils.Secp256k1PrivateKey `json:"privateKey"`
}

// TODO : Update password param and implement password based keystore
// KeyStore defines the interface for managing private keys
type KeyStore interface {
	// Store saves a key with an alias
	Store(alias string, key *Key) error

	// Load retrieves a key by its alias
	Load(alias string) (*Key, error)

	// Delete removes a key by its alias
	Delete(alias string) error

	// ListByAlias returns a map of alias to Key
	ListByAlias() (map[string]*Key, error)

	// ListByEvmAddress returns a map of EVMAddress to Key
	ListByEvmAddress() (map[common.Address]*Key, error)

	// ListByElderAddress returns a map of ElderAddress to Key
	ListByElderAddress() (map[string]*Key, error)
}

func writeContentToFile(file string, content []byte) error {
	name, err := writeTemporaryKeyFile(file, content)
	if err != nil {
		return err
	}

	return os.Rename(name, file)
}

func writeTemporaryKeyFile(file string, content []byte) (string, error) {
	// Create the keystore directory with appropriate permissions
	if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
		return "", fmt.Errorf("failed to create keystore directory: %w", err)
	}

	// Create temporary file with pattern
	f, err := os.CreateTemp(filepath.Dir(file), "."+filepath.Base(file)+".tmp")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %w", err)
	}

	defer func() {
		f.Close()
		// Remove the temporary file if any error occurred during write
		if err != nil {
			os.Remove(f.Name())
		}
	}()

	if _, err := f.Write(content); err != nil {
		return "", fmt.Errorf("failed to write content: %w", err)
	}

	// Ensure all data is written to disk
	if err := f.Sync(); err != nil {
		return "", fmt.Errorf("failed to sync file: %w", err)
	}

	return f.Name(), nil
}
