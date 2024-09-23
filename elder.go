package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"math"
	"os"

	"time"

	"github.com/0xElder/elder/x/router/keeper"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/grpc/cmtservice"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"golang.org/x/crypto/ripemd160"
	"google.golang.org/grpc"

	cosmosmath "cosmossdk.io/math"
	elderregistration "github.com/0xElder/elder/api/elder/registration"
	bech32 "github.com/btcsuite/btcutil/bech32"
)

var ElderNonceMap = make(map[string]uint64)

func BuildElderTxFromMsgAndBroadcast(conn *grpc.ClientConn, msg sdktypes.Msg) error {
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(interfaceRegistry)

	// Create a new transaction builder
	txConfig := authtx.NewTxConfig(cdc, authtx.DefaultSignModes)
	txBuilder := txConfig.NewTxBuilder()

	err := txBuilder.SetMsgs(msg)
	if err != nil {
		log.Fatalf("Failed to set message: %v", err)
		return err
	}

	// Sign the transaction
	txBytes, err := signTx(conn, txConfig, txBuilder, true)
	if err != nil {
		log.Fatalf("Failed to sign the transaction: %v", err)
		return err
	}

	// Simulate the transaction to estimate gas
	gasEstimate, err := simulateElderTx(conn, txBytes)
	if err != nil {
		log.Fatalf("Failed to simulate the transaction: %v", err)
		return err
	}

	// Apply a gas adjustment (e.g., 1.3 to add 30% buffer)
	gasAdjustment := 1.3
	adjustedGas := uint64(float64(gasEstimate) * gasAdjustment)

	// Set gas price
	gp := os.Getenv("ELDER_GAS_PRICE")
	var gasPrice float64
	if gp == "" {
		// default gas price
		gasPrice = .1 * math.Pow(10, -6) // .1 uelder/gas
	}

	// Set a fee amount
	feeAmount := cosmosmath.NewInt(int64(math.Ceil((float64(adjustedGas) * gasPrice))))
	fee := sdktypes.NewCoin("elder", feeAmount)

	// Set the gas limit and fee amount in txBuilder
	txBuilder.SetGasLimit(adjustedGas)
	txBuilder.SetFeeAmount(sdktypes.NewCoins(fee))

	// Sign the transaction
	txBytes, err = signTx(conn, txConfig, txBuilder, false)
	if err != nil {
		log.Fatalf("Failed to sign the transaction: %v", err)
		return err
	}

	// Broadcast the transaction
	err = broadcastElderTx(conn, txBytes)
	if err != nil {
		log.Fatalf("Failed to broadcast the transaction: %v", err)
		return err
	}

	return nil
}

func signTx(conn *grpc.ClientConn, txConfig client.TxConfig, txBuilder client.TxBuilder, simulate bool) ([]byte, error) {
	elderAddress := CosmosPublicKeyToCosmosAddress("elder", hex.EncodeToString(privateKey.PubKey().Bytes()))
	// Account and sequence number: Fetch this from your chain (e.g., using gRPC)
	accountNumber, sequenceNumber, err := queryElderAccount(conn, elderAddress)
	if err != nil {
		log.Fatalf("Failed to fetch account info: %v", err)
		return []byte{}, err
	}

	// If we are using the tx to simulate then don't update the map with the nonce
	if nonceFromMap, ok := ElderNonceMap[elderAddress]; ok {
		sequenceNumber = nonceFromMap
	} else if simulate == false {
		ElderNonceMap[elderAddress] = sequenceNumber + uint64(1)
	}

	chainId := queryElderChainID(conn)

	signerData := authsigning.SignerData{
		ChainID:       chainId,
		AccountNumber: accountNumber,
		Sequence:      sequenceNumber,
		PubKey:        privateKey.PubKey(),
		Address:       privateKey.PubKey().Address().String(),
	}

	signatureV2 := signing.SignatureV2{
		PubKey: privateKey.PubKey(),
		Data: &signing.SingleSignatureData{
			SignMode:  signing.SignMode(txConfig.SignModeHandler().DefaultMode()),
			Signature: nil,
		},
		Sequence: sequenceNumber,
	}
	err = txBuilder.SetSignatures(signatureV2)
	if err != nil {
		log.Fatalf("Failed to set signatures: %v", err)
		return []byte{}, err
	}

	// Sign the transaction
	signatureV2, err = tx.SignWithPrivKey(
		context.Background(),
		signing.SignMode(txConfig.SignModeHandler().DefaultMode()),
		signerData,
		txBuilder,
		&privateKey,
		txConfig,
		sequenceNumber,
	)
	if err != nil {
		log.Fatalf("Failed to sign the transaction: %v", err)
		return []byte{}, err
	}

	err = txBuilder.SetSignatures(signatureV2)
	if err != nil {
		log.Fatalf("Failed to set signatures: %v", err)
		return []byte{}, err
	}

	// Encode the transaction
	txBytes, err := txConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		log.Fatalf("Failed to encode the transaction: %v", err)
		return []byte{}, err
	}

	return txBytes, nil
}

func queryElderChainID(conn *grpc.ClientConn) string {
	// Create a client for querying the Tendermint chain
	tmClient := cmtservice.NewServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	status, err := tmClient.GetNodeInfo(ctx, &cmtservice.GetNodeInfoRequest{})
	if err != nil {
		log.Fatalf("Failed to fetch chain info: %v", err)
	}

	fmt.Printf("Chain ID: %s\n", status.DefaultNodeInfo.Network)
	return status.DefaultNodeInfo.Network
}

func queryElderAccount(conn *grpc.ClientConn, address string) (uint64, uint64, error) {
	// Create a client for querying account data
	authClient := authtypes.NewQueryClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	// Fetch the account information
	accountReq := &authtypes.QueryAccountRequest{
		Address: address,
	}
	accountRes, err := authClient.Account(ctx, accountReq)
	if err != nil {
		log.Fatalf("Failed to fetch account info: %v", err)
		return 0, 0, err
	}

	// Unmarshal the account info
	var account authtypes.BaseAccount
	err = account.Unmarshal(accountRes.Account.Value)
	if err != nil {
		log.Fatalf("Failed to unmarshal account info: %v", err)
		return 0, 0, err
	}

	fmt.Printf("Account Number: %d, Sequence: %d\n", account.AccountNumber, account.Sequence)
	return account.AccountNumber, account.Sequence, nil
}

func queryElderRollMinTxFees(conn *grpc.ClientConn, rollId uint64) (uint64, error) {
	// Create a client for querying the roll registration
	registerClient := elderregistration.NewQueryClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	// Fetch the roll registration
	rollReq := &elderregistration.QueryQueryRollRequest{
		Id: rollId,
	}
	rollRes, err := registerClient.QueryRoll(ctx, rollReq)
	if err != nil {
		log.Fatalf("Failed to fetch roll registration: %v", err)
		return 0, err
	}

	return rollRes.Roll.MinTxFees, nil
}

func broadcastElderTx(conn *grpc.ClientConn, txBytes []byte) error {
	// Broadcast the tx via gRPC. We create a new client for the Protobuf Tx
	// service.
	txClient := txtypes.NewServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	// We then call the BroadcastTx method on this client.
	grpcRes, err := txClient.BroadcastTx(
		ctx,
		&txtypes.BroadcastTxRequest{
			Mode:    txtypes.BroadcastMode_BROADCAST_MODE_SYNC,
			TxBytes: txBytes, // Proto-binary of the signed transaction, see previous step.
		},
	)
	if err != nil {
		return err
	}

	fmt.Println(grpcRes.TxResponse)
	return nil
}

func simulateElderTx(conn *grpc.ClientConn, txBytes []byte) (uint64, error) {
	// Simulate the tx via gRPC. We create a new client for the Protobuf Tx
	// service.
	txClient := txtypes.NewServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	// We then call the SimulateTx method on this client.
	grpcRes, err := txClient.Simulate(
		ctx,
		&txtypes.SimulateRequest{
			TxBytes: txBytes, // Proto-binary of the signed transaction, see previous step.
		},
	)
	if err != nil {
		return 0, err
	}

	fmt.Println(grpcRes.GasInfo.GasUsed)
	return grpcRes.GasInfo.GasUsed, nil
}
func calcTxFees(conn *grpc.ClientConn, txData []byte, rollId uint64) uint64 {
	// Fetch the fees per byte from the chain
	feesPerByte, err := queryElderRollMinTxFees(conn, rollId)
	if err != nil {
		return 0
	}

	return keeper.TxFees(txData, feesPerByte)
}

// PublicKeyToAddress converts secp256k1 public key to a bech32 Tendermint/Cosmos based address
func CosmosPublicKeyToCosmosAddress(addressPrefix, publicKeyString string) string {
	// Decode public key string
	pubKeyBytes, err := hex.DecodeString(publicKeyString)
	if err != nil {
		log.Fatalf("Failed to decode public key hex: %v", err)
	}

	// Hash pubKeyBytes as: RIPEMD160(SHA256(public_key_bytes))
	pubKeySha256Hash := sha256.Sum256(pubKeyBytes)
	ripemd160hash := ripemd160.New()
	ripemd160hash.Write(pubKeySha256Hash[:])
	addressBytes := ripemd160hash.Sum(nil)

	// Convert addressBytes into a bech32 string
	address := toBech32(addressPrefix, addressBytes)

	return address
}

// Code courtesy: https://github.com/cosmos/cosmos-sdk/blob/90c9c9a9eb4676d05d3f4b89d9a907bd3db8194f/types/bech32/bech32.go#L10
func toBech32(addrPrefix string, addrBytes []byte) string {
	converted, err := bech32.ConvertBits(addrBytes, 8, 5, true)
	if err != nil {
		panic(err)
	}

	addr, err := bech32.Encode(addrPrefix, converted)
	if err != nil {
		panic(err)
	}

	return addr
}
