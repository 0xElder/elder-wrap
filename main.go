package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"net/http"

	"github.com/0xElder/elder/x/router/types"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Target RPC endpoint to forward the requests

var privateKey secp256k1.PrivKey
var rollID uint64
var elderRpc string
var rollAppRpc string

// Middleware to handle and relay the JSON-RPC requests
func rpcHandler(w http.ResponseWriter, r *http.Request) {
	// body, err := io.ReadAll(r.Body)
	// if err != nil {
	// 	http.Error(w, "Failed to read request", http.StatusBadRequest)
	// 	return
	// }

	// var rpcRequest JsonRPCRequest
	// err = json.Unmarshal(body, &rpcRequest)
	// if err != nil {
	// 	http.Error(w, "Invalid JSON-RPC request", http.StatusBadRequest)
	// 	return
	// }

	// // Check if the method is `eth_sendRawTransaction` (signed transaction)
	// // if rpcRequest.Method == "eth_sendRawTransaction" || rpcRequest.Method == "eth_sendTransaction" {
	// log.Println("Caught a signed transaction:", rpcRequest)

	// internalTx, ok := rpcRequest.Params[0].(string)
	// if !ok {
	// 	http.Error(w, "Invalid transaction", http.StatusBadRequest)
	// 	return
	// }

	internalTx := "anshal"
	// VerifyReceivedRollAppTx(rollAppRpc, internalTx)
	internalTxBytes := []byte(internalTx)

	elderAddress := CosmosPublicKeyToCosmosAddress("elder", hex.EncodeToString(privateKey.PubKey().Bytes()))
	// elderAddress := "elder1rrr78xn2gkj4d5tpc9cpuydadvm94xwg0a2hcj"
	msg := &types.MsgSubmitRollTx{
		RollId:       1,
		TxData:       internalTxBytes,
		MaxFeesGiven: uint64(100),
		Sender:       elderAddress,
	}

	conn, err := grpc.NewClient(elderRpc, grpc.WithTransportCredentials(insecure.NewCredentials())) // The Cosmos SDK doesn't support any transport security mechanism.
	if err != nil {
		http.Error(w, "Failed to connect to the elder RPC", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	response := JsonRPCResponse{
		JsonRPC: "2.0",
		ID:      1,
	}

	// Build the transaction and broadcast it
	err = BuildElderTxFromMsgAndBroadcast(conn, msg)
	if err != nil {
		log.Fatalf("Failed to build and broadcast transaction: %v", err)
		response.Error = err.Error()
		response.Result = "{'status': '0x0'}"
	}
	response.Error = nil
	response.Result = "{'status': '0x1'}"

	// Send the response back
	// w.Header().Set("Content-Type", "application/json")
	// json.NewEncoder(w).Encode(response)

	// } else {
	// 	// Relay all other RPC calls to rollup RPC
	// 	ForwardToRollAppRPC(w, rollAppRpc, body)
	// }
}

func main() {
	// // Validate if all the environment variables are set
	// requiredEnvVars := []string{"ELDER_RPC", "ROLL_APP_RPC", "COSMOS_PRIVATE_KEY"}
	// for _, envVar := range requiredEnvVars {
	// 	if len(envVar) == 0 {
	// 		log.Fatalf("Please set the environment variable %s", envVar)
	// 	}
	// }

	// // Set global variables
	// elderRpc = os.Getenv("ELDER_RPC")
	// rollAppRpc = os.Getenv("ROLL_APP_RPC")

	// // Get rollup ID
	// // Just to make sure the RPC is working
	// _, err := GetRollAppId(rollAppRpc)
	// if err != nil {
	// 	log.Fatalf("Failed to fetch rollup ID: %v", err)
	// 	return
	// }

	// Set up the public/private key
	privateKeyBytes, err := hex.DecodeString("424dc4f4f82c716a3d17a2753f1a044cd1112071ce30595f9abae2010ed389aa")
	fmt.Println("privateKeyBytes: ", privateKeyBytes)
	fmt.Println("err: ", err)
	if err != nil {
		log.Fatalf("Failed to decode private key: %v", err)
	}

	// Load the SECP256K1 private key from the decoded bytes
	privKey, _ := btcec.PrivKeyFromBytes(privateKeyBytes)

	// Generate a Cosmos-compatible SECP256K1 private key
	// cosmosPrivKey := secp256k1.PrivKey{Key: privKey.Serialize()}

	privateKey = secp256k1.PrivKey{
		Key: privKey.Serialize(),
	}

	fmt.Println("privateKey: ", privateKey.PubKey())
	// Setup the HTTP server, listening on port 8545
	// http.HandleFunc("/", rpcHandler)
	// fmt.Println("Starting server on port 8545")
	// log.Fatal(http.ListenAndServe(":8545", nil))

	elderRpc = "192.168.1.6:9090"
	rpcHandler(nil, nil)
}
