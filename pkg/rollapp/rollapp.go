package rollapp

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"

	"github.com/0xElder/elder-wrap/pkg/keystore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"google.golang.org/grpc"
)

type RollApp struct {
	RPC                string
	ElderRegistationId uint64
	client             *ethclient.Client
	keyStore           keystore.KeyStore
	elderConn          *grpc.ClientConn
}

func NewRollApp(rpc string, elderId uint64, keyStore keystore.KeyStore, elderConn *grpc.ClientConn) (*RollApp, error) {
	client, err := ethclient.Dial(rpc)
	if err != nil {
		return nil, err
	}

	return &RollApp{
		RPC:                rpc,
		ElderRegistationId: elderId,
		client:             client,
		keyStore:           keyStore,
		elderConn:          elderConn,
	}, nil
}

func (r *RollApp) GetRollAppId(ctx context.Context) (uint64, error) {
	id, err := r.client.ChainID(ctx)
	if err != nil {
		return 0, err
	}
	return id.Uint64(), nil
}

func (r *RollApp) GetAddressNonce(ctx context.Context, address string) (uint64, error) {
	nonce, err := r.client.PendingNonceAt(ctx, common.HexToAddress(address))
	if err != nil {
		return 0, err
	}
	return nonce, nil
}

func (r *RollApp) VerifyRollAppTx(ctx context.Context, rawTx string) (*types.Transaction, *keystore.Key, error) {
	txBytes, err := hex.DecodeString(rawTx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decode tx: %w", err)
	}

	var tx types.Transaction
	err = tx.UnmarshalBinary(txBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal tx: %w", err)
	}

	txChainId := tx.ChainId()
	chainIdRPC, err := r.GetRollAppId(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get chain id: %w", err)
	}

	if txChainId.Uint64() != chainIdRPC {
		return nil, nil, fmt.Errorf("chain id mismatch: expected %d, got %d", chainIdRPC, tx.ChainId().Uint64())
	}

	fromAddress, err := types.LatestSignerForChainID(txChainId).Sender(&tx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get from address: %w", err)
	}

	KeyListByEvmAddress, err := r.keyStore.ListByEvmAddress()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch keys: %w", err)
	}

	key, ok := KeyListByEvmAddress[fromAddress]
	if !ok {
		return nil, nil, fmt.Errorf("key not found: %s", fromAddress.Hex())
	}

	address := key.EvmAddress
	if fromAddress.Cmp(address) != 0 {
		return nil, nil, fmt.Errorf("address mismatch: expected %s, got %s", address.Hex(), fromAddress.Hex())
	}

	nonce := tx.Nonce()
	nonceRPC, err := r.GetAddressNonce(ctx, fromAddress.Hex())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get nonce: %w", err)
	}

	if nonce != nonceRPC {
		return nil, nil, fmt.Errorf("nonce mismatch: expected %d, got %d", nonceRPC, nonce)
	}

	return &tx, key, nil
}

func (r *RollApp) ForwardtoRollAppRPC(w http.ResponseWriter, body []byte) {
	// Forward the request to the rollApp RPC endpoint
	resp, err := http.Post(r.RPC, "application/json", bytes.NewBuffer(body))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Read the response from the rollApp RPC
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Write the response to the client
	w.Header().Set("Content-Type", "application/json")
	w.Write(responseBody)
}
