package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/0xElder/elder/x/router/types"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Global variables
var privateKey secp256k1.PrivKey
var rollId uint64
var elderGrpc string
var rollAppRpc string
var elderAddress string

// Middleware to handle and relay the JSON-RPC requests
func rpcHandler(w http.ResponseWriter, r *http.Request) {
	// Set the response header
	w.Header().Set("Content-Type", "application/json")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request", http.StatusBadRequest)
		return
	}

	var rpcRequest JsonRPCRequest
	err = json.Unmarshal(body, &rpcRequest)
	if err != nil {
		http.Error(w, "Invalid JSON-RPC request", http.StatusBadRequest)
		return
	}

	// Check if the method is `eth_sendRawTransaction` (signed transaction)
	if rpcRequest.Method == "eth_sendRawTransaction" {
		response := JsonRPCResponse{
			JsonRPC: rpcRequest.JsonRPC,
			ID:      rpcRequest.ID,
		}

		// Send the response back
		defer func() {
			err := json.NewEncoder(w).Encode(response)
			if err != nil {
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			}
		}()

		log.Println("Caught a signed transaction:", rpcRequest)

		internalTx, ok := rpcRequest.Params[0].(string)
		if !ok {
			response.Error = fmt.Errorf("invalid transaction format")
			return
		}

		if internalTx[0:2] != "0x" {
			internalTx = "0x" + internalTx
		}

		tx, err := VerifyReceivedRollAppTx(rollAppRpc, internalTx[2:])
		if err != nil {
			response.Error = err.Error()
			return
		}

		internalTxBytes, err := hexutil.Decode(internalTx)
		if err != nil {
			response.Error = err.Error()
			return
		}

		conn, err := grpc.NewClient(elderGrpc, grpc.WithTransportCredentials(insecure.NewCredentials())) // The Cosmos SDK doesn't support any transport security mechanism.
		if err != nil {
			response.Error = err.Error()
			return
		}
		defer conn.Close()

		msg := &types.MsgSubmitRollTx{
			RollId:       rollId,
			TxData:       internalTxBytes,
			MaxFeesGiven: calcTxFees(conn, internalTxBytes, rollId),
			Sender:       elderAddress,
		}

		// Build the transaction and broadcast it
		elderTxHash, err := BuildElderTxFromMsgAndBroadcast(conn, msg)
		if elderTxHash == "" || err != nil {
			response.Error = fmt.Errorf("failed to broadcast transaction, elderTxHash: %v, err: %v", elderTxHash, err)
			return
		}

		_, rollAppBlock, err := getElderTxFromHash(conn, elderTxHash)
		if err != nil || rollAppBlock == "" {
			response.Error = fmt.Errorf("failed to fetch elder tx, rollAppBlock: %v, err: %v", rollAppBlock, err)
			return
		}

		response.Result = tx.Hash().String()
	} else {
		// Relay all other RPC calls to rollup RPC
		ForwardToRollAppRPC(w, rollAppRpc, body)
	}
}

func main() {
	// Validate if all the environment variables are set
	requiredEnvVars := []string{"ELDER_RPC", "ROLL_ID", "ROLL_APP_RPC", "COSMOS_PRIVATE_KEY", "PORT"}
	for _, envVar := range requiredEnvVars {
		if len(envVar) == 0 {
			log.Fatalf("Please set the environment variable %s\n", envVar)
		}
	}

	// Set global variables
	elderGrpc = os.Getenv("ELDER_gRPC")
	rollAppRpc = os.Getenv("ROLL_APP_RPC")
	rollIdStr := os.Getenv("ROLL_ID")
	portStr := os.Getenv("PORT")

	rollIdStr = strings.TrimPrefix(rollIdStr, "http://")
	rollIdStr = strings.TrimPrefix(rollIdStr, "https://")

	var err error
	rollId, err = strconv.ParseUint(rollIdStr, 10, 64)
	if err != nil {
		log.Fatalf("Failed to parse roll ID: %v\n", err)
		return
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		log.Fatalf("Failed to parse port: %v\n", err)
		return
	}

	// Get rollup ID
	// Just to make sure the RPC is working
	_, err = GetRollAppId(rollAppRpc)
	if err != nil {
		log.Fatalf("Failed to fetch rollup ID: %v\n", err)
		return
	}

	// Set up the public/private key
	pk_env := os.Getenv("COSMOS_PRIVATE_KEY")
	if pk_env[0:2] == "0x" {
		pk_env = pk_env[2:]
	}

	pkBytes, err := hex.DecodeString(pk_env)
	if err != nil {
		log.Fatalf("Failed to decode private key: %v\n", err)
	}

	// Load the SECP256K1 private key from the decoded bytes
	pk, _ := btcec.PrivKeyFromBytes(pkBytes)
	privateKey = secp256k1.PrivKey{
		Key: pk.Serialize(),
	}

	// Get the elder address
	elderAddress = CosmosPublicKeyToCosmosAddress("elder", hex.EncodeToString(privateKey.PubKey().Bytes()))
	log.Printf("Elder address: %s\n", elderAddress)

	http.HandleFunc("/elder-address", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(elderAddress)
	})

	// Setup the HTTP server, listening on port 8546
	http.HandleFunc("/", rpcHandler)
	fmt.Printf("Starting server on port %d\n", port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
