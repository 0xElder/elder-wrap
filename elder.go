package main

import (
	"context"
	"fmt"
	"log"

	"time"

	"github.com/0xElder/elder/x/router/keeper"
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
	"google.golang.org/grpc"
)

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

	// Set the gas limit
	// This is a random number 200000
	txBuilder.SetGasLimit(200000)
	// txBuilder.SetFeeAmount()
	// txBuilder.SetMemo()
	// txBuilder.SetTimeoutHeight()

	// Account and sequence number: Fetch this from your chain (e.g., using gRPC)
	accountNumber, sequenceNumber, err := queryElderAccount(conn, privateKey.PubKey().Address().String())
	if err != nil {
		log.Fatalf("Failed to fetch account info: %v", err)
		return err
	}

	chainId := queryElderChainID(conn)

	signerData := authsigning.SignerData{
		ChainID:       chainId,
		AccountNumber: accountNumber,
		Sequence:      sequenceNumber,
	}

	// Sign the transaction
	signatureV2, err := tx.SignWithPrivKey(
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
		return err
	}

	err = txBuilder.SetSignatures(signatureV2)
	if err != nil {
		log.Fatalf("Failed to set signatures: %v", err)
		return err
	}

	// Encode the transaction
	txBytes, err := txConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		log.Fatalf("Failed to encode the transaction: %v", err)
		return err
	}

	fmt.Println("txBytes: ", txBytes)

	// Broadcast the transaction
	err = broadcastElderTx(conn, txBytes)
	if err != nil {
		log.Fatalf("Failed to broadcast the transaction: %v", err)
		return err
	}

	return nil
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

	fmt.Println(grpcRes.TxResponse.Code)
	return nil
}

func calcTxFees(txData []byte) uint64 {
	// Random number 5
	feesPerByte := uint64(5)
	keeper.TxFees(txData, feesPerByte)
	return 0
}
