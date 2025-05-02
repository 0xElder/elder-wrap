package keystore

import (
	"github.com/0xElder/elder-wrap/pkg/logging"
	"github.com/0xElder/elder/app/constants"
	"github.com/0xElder/elder/utils"
	"github.com/ethereum/go-ethereum/common"

	"github.com/pkg/errors"
)

type KeyStoreClient struct {
	store  KeyStore
	logger logging.Logger
}

func NewKeyStoreClient(store KeyStore, logger logging.Logger) *KeyStoreClient {
	return &KeyStoreClient{store: store, logger: logger}
}

// ImportPrivateKey imports a private key with an alias
func (s *KeyStoreClient) ImportPrivateKey(alias string, privateKeyHex string) error {
	s.logger.Debug(nil, "Importing private key", "alias", alias, "privateKeyHex", privateKeyHex)
	privateKey, err := utils.PrivateKeyStringToSecp256k1PrivKey(privateKeyHex)
	if err != nil {
		s.logger.Error(nil, "Failed to import private key", "error", err)
		return errors.Wrap(err, "failed to import private key")
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
	s.logger.Debug(nil, "Deleting key", "alias", alias)
	return s.store.Delete(alias)
}

// ListKeys returns all stored keys by their aliases
func (s *KeyStoreClient) ListKeys() (map[string]*Key, error) {
	s.logger.Debug(nil, "Listing keys")
	return s.store.ListByAlias()
}

// GetKeyByAlias retrieves a key by its alias
func (s *KeyStoreClient) GetKeyByAlias(alias string) (*Key, error) {
	s.logger.Debug(nil, "Getting key by alias", "alias", alias)
	keys, err := s.store.ListByAlias()
	if err != nil {
		s.logger.Error(nil, "Failed to list keys by alias", "error", err)
		return nil, errors.Wrap(err, "failed to list keys by alias")
	}
	key, exists := keys[alias]
	if !exists {
		s.logger.Error(nil, "Key not found", "alias", alias)
		return nil, errors.New("key not found")
	}
	return key, nil
}

// GetKeyByEvmAddress retrieves a key by its EVM address
func (s *KeyStoreClient) GetKeyByEvmAddress(address string) (*Key, error) {
	s.logger.Debug(nil, "Getting key by EVM address", "address", address)
	keys, err := s.store.ListByEvmAddress()
	if err != nil {
		s.logger.Error(nil, "Failed to list keys by EVM address", "error", err)
		return nil, errors.Wrap(err, "failed to list keys by EVM address")
	}

	addr := common.HexToAddress(address)
	key, exists := keys[addr]
	if !exists {
		s.logger.Error(nil, "Key not found", "address", address)
		return nil, errors.New("key not found")
	}

	return key, nil
}

// GetKeyByElderAddress retrieves a key by its Elder address
func (s *KeyStoreClient) GetKeyByElderAddress(address string) (*Key, error) {
	s.logger.Debug(nil, "Getting key by Elder address", "address", address)
	keys, err := s.store.ListByElderAddress()
	if err != nil {
		s.logger.Error(nil, "Failed to list keys by Elder address", "error", err)
		return nil, errors.Wrap(err, "failed to list keys by Elder address")
	}

	key, exists := keys[address]
	if !exists {
		s.logger.Error(nil, "Key not found", "address", address)
		return nil, errors.New("key not found")
	}

	return key, nil
}

func privateKeyToElderAddress(privateKey utils.Secp256k1PrivateKey) string {
	return utils.CosmosPublicKeyToBech32Address(constants.Bech32PrefixAccAddr, privateKey.PubKey())
}
