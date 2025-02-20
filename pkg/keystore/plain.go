package keystore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

// PlainKeyStore implements the KeyStore interface using a plain file system without password
type PlainKeyStore struct {
	baseDir string
	mu      sync.RWMutex
}

func NewPlainKeyStore(baseDir string) (*PlainKeyStore, error) {
	if err := os.MkdirAll(baseDir, 0700); err != nil {
		return nil, err
	}
	return &PlainKeyStore{
		baseDir: baseDir,
	}, nil
}

func (ks *PlainKeyStore) Store(alias string, key *Key) error {
	if len(alias) == 0 {
		return fmt.Errorf("alias cannot be empty")
	}
	if key == nil {
		return fmt.Errorf("key cannot be nil")
	}

	ks.mu.Lock()
	defer ks.mu.Unlock()

	keyPath := ks.joinPath(alias)

	// Check if file exists
	if _, err := os.Stat(keyPath); err == nil {
		return fmt.Errorf("key %s: %w", alias, ErrKeyExists)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check key file: %w", err)
	}

	// Marshal key data
	content, err := json.Marshal(key)
	if err != nil {
		return fmt.Errorf("failed to marshal key data: %w", err)
	}

	return writeContentToFile(keyPath, content)
}

func (ks *PlainKeyStore) Load(alias string) (*Key, error) {
	if len(alias) == 0 {
		return nil, fmt.Errorf("alias cannot be empty")
	}

	ks.mu.RLock()
	defer ks.mu.RUnlock()

	keyPath := ks.joinPath(alias)

	file, err := os.Open(keyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("key %s: %w", alias, ErrKeyNotFound)
		}
		return nil, fmt.Errorf("failed to open key file: %w", err)
	}
	defer file.Close()

	var key Key
	if err := json.NewDecoder(file).Decode(&key); err != nil {
		return nil, fmt.Errorf("failed to decode key data: %w", err)
	}

	return &key, nil
}

func (ks *PlainKeyStore) Delete(alias string) error {
	if len(alias) == 0 {
		return fmt.Errorf("alias cannot be empty")
	}

	ks.mu.Lock()
	defer ks.mu.Unlock()

	keyPath := ks.joinPath(alias)
	if err := os.Remove(keyPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("key %s: %w", alias, ErrKeyNotFound)
		}
		return fmt.Errorf("failed to delete key file: %w", err)
	}
	return nil
}

func (ks *PlainKeyStore) ListByAlias() (map[string]*Key, error) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	files, err := os.ReadDir(ks.baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	result := make(map[string]*Key)
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" {
			alias := strings.TrimSuffix(file.Name(), ".json")
			// Read file directly instead of using Load to avoid double locking
			keyPath := ks.joinPath(alias)

			file, err := os.Open(keyPath)
			if err != nil {
				return nil, fmt.Errorf("failed to open key %s: %w", alias, err)
			}

			var key Key
			err = json.NewDecoder(file).Decode(&key)
			file.Close()
			if err != nil {
				return nil, fmt.Errorf("failed to decode key %s: %w", alias, err)
			}

			result[alias] = &key
		}
	}
	return result, nil
}

func (ks *PlainKeyStore) ListByEvmAddress() (map[common.Address]*Key, error) {
	keys, err := ks.ListByAlias()
	if err != nil {
		return nil, err
	}

	result := make(map[common.Address]*Key)
	for _, key := range keys {
		result[key.EvmAddress] = key
	}
	return result, nil
}

func (ks *PlainKeyStore) ListByElderAddress() (map[string]*Key, error) {
	keys, err := ks.ListByAlias()
	if err != nil {
		return nil, err
	}

	result := make(map[string]*Key)
	for _, key := range keys {
		result[key.ElderAddress] = key
	}
	return result, nil
}

func (ks *PlainKeyStore) joinPath(alias string) string {
	return filepath.Join(ks.baseDir, alias+".json")
}
