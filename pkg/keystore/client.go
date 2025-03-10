package keystore

import (
	"fmt"

	"github.com/0xElder/elder/utils"
	"github.com/ethereum/go-ethereum/common"
)

type KeyStoreClient struct {
	store KeyStore
}

func NewKeyStoreClient(store KeyStore) *KeyStoreClient {
	return &KeyStoreClient{store: store}
}

// ImportPrivateKey imports a private key with an alias
func (s *KeyStoreClient) ImportPrivateKey(alias string, privateKeyHex string) error {
	privateKey, err := utils.PrivateKeyStringToSecp256k1PrivKey(privateKeyHex)
	if err != nil {
		return fmt.Errorf("failed to import private key: %v", err)
	}

	key := &Key{
		EvmAddress:   common.BytesToAddress(privateKey.PubKey().Address()),
		ElderAddress: privateKeyToElderAddress(privateKey),
		PrivateKey:   privateKey,
	}

	return s.store.Store(alias, key)
}

// DeleteKey removes a key by its alias
func (s *KeyStoreClient) DeleteKey(alias string) error {
	return s.store.Delete(alias)
}

// ListKeys returns all stored keys by their aliases
func (s *KeyStoreClient) ListKeys() (map[string]*Key, error) {
	return s.store.ListByAlias()
}

// GetKeyByAlias retrieves a key by its alias
func (s *KeyStoreClient) GetKeyByAlias(alias string) (*Key, error) {
	keys, err := s.store.ListByAlias()
	if err != nil {
		return nil, err
	}
	key, exists := keys[alias]
	if !exists {
		return nil, fmt.Errorf("key with alias %s not found", alias)
	}
	return key, nil
}

// GetKeyByEvmAddress retrieves a key by its EVM address
func (s *KeyStoreClient) GetKeyByEvmAddress(address string) (*Key, error) {
	keys, err := s.store.ListByEvmAddress()
	if err != nil {
		return nil, err
	}

	addr := common.HexToAddress(address)
	key, exists := keys[addr]
	if !exists {
		return nil, fmt.Errorf("key with EVM address %s not found", address)
	}

	return key, nil
}

// GetKeyByElderAddress retrieves a key by its Elder address
func (s *KeyStoreClient) GetKeyByElderAddress(address string) (*Key, error) {
	keys, err := s.store.ListByElderAddress()
	if err != nil {
		return nil, err
	}

	key, exists := keys[address]
	if !exists {
		return nil, fmt.Errorf("key with Elder address %s not found", address)
	}

	return key, nil
}

func privateKeyToElderAddress(privateKey utils.Secp256k1PrivateKey) string {
	return utils.CosmosPublicKeyToBech32Address("elder", privateKey.PubKey())
}
