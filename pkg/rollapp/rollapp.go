package rollapp

import (
	"bytes"
	"context"
	"encoding/hex"
	"io"
	"net/http"

	"github.com/pkg/errors"

	"github.com/0xElder/elder-wrap/pkg/elder"
	"github.com/0xElder/elder-wrap/pkg/keystore"
	"github.com/0xElder/elder-wrap/pkg/logging"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum/go-ethereum/ethclient"
)

type RollApp struct {
	RPC                string
	ElderRegistationId uint64
	client             *ethclient.Client
	logger             logging.Logger
	keyStore           keystore.KeyStore
	elderClient        *elder.ElderClient
}

func NewRollApp(rpc string, elderId uint64, keyStore keystore.KeyStore, logger logging.Logger, elderClient *elder.ElderClient) (*RollApp, error) {
	client, err := ethclient.Dial(rpc)
	if err != nil {
		return nil, err
	}

	return &RollApp{
		RPC:                rpc,
		ElderRegistationId: elderId,
		client:             client,
		logger:             logger,
		keyStore:           keyStore,
		elderClient:        elderClient,
	}, nil
}

func (r *RollApp) GetRollAppId(ctx context.Context) (uint64, error) {
	logger := r.logger.With("method", "GetRollAppId")
	logger.Debug(ctx, "Fetching chain ID from rollapp RPC")
	id, err := r.client.ChainID(ctx)
	if err != nil {
		return 0, err
	}
	logger.Debug(ctx, "Fetched chain ID from rollapp RPC", "chainId", id.Uint64())
	return id.Uint64(), nil
}

func (r *RollApp) GetAddressNonce(ctx context.Context, address string) (uint64, error) {
	logger := r.logger.With("method", "GetAddressNonce")
	logger.Debug(ctx, "Fetching nonce for address", "address", address)
	nonce, err := r.client.PendingNonceAt(ctx, common.HexToAddress(address))
	if err != nil {
		return 0, err
	}
	logger.Debug(ctx, "Fetched nonce for address", "address", address, "nonce", nonce)
	return nonce, nil
}

func (r *RollApp) VerifyRollAppTx(ctx context.Context, rawTx string) (*types.Transaction, *keystore.Key, error) {
	logger := r.logger.With("method", "VerifyRollAppTx")
	logger.Debug(ctx, "Verifying rollapp transaction", "rawTx", rawTx)
	txBytes, err := hex.DecodeString(rawTx)
	if err != nil {
		logger.Error(ctx, "Failed to decode raw transaction", "error", err)
		return nil, nil, errors.Wrap(err, "failed to decode raw transaction")
	}

	var tx types.Transaction
	err = tx.UnmarshalBinary(txBytes)
	if err != nil {
		logger.Error(ctx, "Failed to unmarshal transaction", "error", err)
		return nil, nil, errors.Wrap(err, "failed to unmarshal transaction")
	}

	txChainId := tx.ChainId()
	chainIdRPC, err := r.GetRollAppId(ctx)
	if err != nil {
		logger.Error(ctx, "Failed to get chain id", "error", err)
		return nil, nil, errors.Wrap(err, "failed to get chain id")
	}

	if txChainId.Uint64() != chainIdRPC {
		logger.Error(ctx, "Chain id mismatch", "expected", chainIdRPC, "got", txChainId.Uint64())
		return nil, nil, errors.New("chain id mismatch")
	}

	fromAddress, err := types.LatestSignerForChainID(txChainId).Sender(&tx)
	if err != nil {
		logger.Error(ctx, "Failed to get sender address", "error", err)
		return nil, nil, errors.Wrap(err, "failed to get sender address")
	}

	KeyListByEvmAddress, err := r.keyStore.ListByEvmAddress()
	if err != nil {
		logger.Error(ctx, "Failed to list keys by EVM address", "error", err)
		return nil, nil, errors.Wrap(err, "failed to list keys by EVM address")
	}

	key, ok := KeyListByEvmAddress[fromAddress]
	if !ok {
		logger.Error(ctx, "Key not found in keystore", "address", fromAddress.Hex())
		return nil, nil, errors.New("key not found in keystore")
	}

	address := key.EvmAddress
	if fromAddress.Cmp(address) != 0 {
		logger.Error(ctx, "Sender address does not match key address", "expected", address.Hex(), "got", fromAddress.Hex())
		return nil, nil, errors.New("sender address does not match key address")
	}

	nonce := tx.Nonce()
	nonceRPC, err := r.GetAddressNonce(ctx, fromAddress.Hex())
	if err != nil {
		logger.Error(ctx, "Failed to get address nonce", "error", err)
		return nil, nil, errors.Wrap(err, "failed to get address nonce")
	}

	if nonce != nonceRPC {
		logger.Error(ctx, "Nonce mismatch", "expected", nonceRPC, "got", nonce)
		return nil, nil, errors.New("nonce mismatch")
	}
	logger.Debug(ctx, "Transaction verified successfully", "rawTx", rawTx, "fromAddress", fromAddress.Hex())
	logger.Debug(ctx, "Transaction details", "chainId", chainIdRPC, "nonce", nonce, "to", tx.To().Hex(), "value", tx.Value().String(), "data", tx.Data())
	return &tx, key, nil
}

func (r *RollApp) ForwardtoRollAppRPC(w http.ResponseWriter, body []byte) {
	logger := r.logger.With("method", "ForwardtoRollAppRPC")
	logger.Debug(context.Background(), "Forwarding request to rollApp RPC", "rpc", r.RPC)
	// Forward the request to the rollApp RPC endpoint
	resp, err := http.Post(r.RPC, "application/json", bytes.NewBuffer(body))
	if err != nil {
		logger.Error(context.Background(), "Failed to forward request to rollApp RPC", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Read the response from the rollApp RPC
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error(context.Background(), "Failed to read response from rollApp RPC", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Write the response to the client
	w.Header().Set("Content-Type", "application/json")
	w.Write(responseBody)
	logger.Debug(context.Background(), "Forwarded response to client", "status", resp.StatusCode, "response", string(responseBody))
}
