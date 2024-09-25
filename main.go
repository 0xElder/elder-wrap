package main

import (
	"encoding/hex"
	"log"
	"net/http"
	"sync"

	"github.com/0xElder/elder/x/router/types"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Global variables
var privateKey secp256k1.PrivKey
var rollId uint64
var elderRpc string
var rollAppRpc string
var elderAddress string

var mutex = sync.Mutex{}

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
	// if rpcRequest.Method == "eth_sendRawTransaction" {
	// 	mutex.Lock()
	// 	defer mutex.Unlock()
	// 	log.Println("Caught a signed transaction:", rpcRequest)

	// 	internalTx, ok := rpcRequest.Params[0].(string)
	// 	if !ok {
	// 		http.Error(w, "Invalid transaction", http.StatusBadRequest)
	// 		return
	// 	}

	// 	VerifyReceivedRollAppTx(rollAppRpc, internalTx)
	internalTx := "anshal"
	internalTxBytes := []byte(internalTx)

	conn, err := grpc.NewClient(elderRpc, grpc.WithTransportCredentials(insecure.NewCredentials())) // The Cosmos SDK doesn't support any transport security mechanism.
	if err != nil {
		http.Error(w, "Failed to connect to the elder RPC", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	msg := &types.MsgSubmitRollTx{
		RollId:       rollId,
		TxData:       internalTxBytes,
		MaxFeesGiven: calcTxFees(conn, internalTxBytes, rollId),
		Sender:       elderAddress,
	}

	response := JsonRPCResponse{
		JsonRPC: "2.0",
		ID:      1,
	}

	// Build the transaction and broadcast it
	err = BuildElderTxFromMsgAndBroadcast(conn, msg)
	if err != nil {
		response.Error = err.Error()
		response.Result = "{'status': '0x1'}"
	}
	response.Error = nil
	response.Result = "{'status': '0x0'}"

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
	// requiredEnvVars := []string{"ELDER_RPC", "ROLL_ID", "ROLL_APP_RPC", "COSMOS_PRIVATE_KEY"}
	// for _, envVar := range requiredEnvVars {
	// 	if len(envVar) == 0 {
	// 		log.Fatalf("Please set the environment variable %s\n", envVar)
	// 	}
	// }

	// // Set global variables
	// elderRpc = os.Getenv("ELDER_RPC")
	// rollAppRpc = os.Getenv("ROLL_APP_RPC")
	// rollIdStr := os.Getenv("ROLL_ID")

	// var err error
	// rollId, err = strconv.ParseUint(rollIdStr, 10, 64)
	// if err != nil {
	// 	log.Fatalf("Failed to parse roll ID: %v\n", err)
	// 	return
	// }

	// // Get rollup ID
	// // Just to make sure the RPC is working
	// _, err = GetRollAppId(rollAppRpc)
	// if err != nil {
	// 	log.Fatalf("Failed to fetch rollup ID: %v\n", err)
	// 	return
	// }

	// // Set up the public/private key
	// pk_env := os.Getenv("COSMOS_PRIVATE_KEY")
	// if pk_env[0:2] == "0x" {
	// 	pk_env = pk_env[2:]
	// }

	pk_env := "e8a419f8d826561245e31adbf2a32158855099e242e71eac8ca199c9967a5679"
	pkBytes, err := hex.DecodeString(pk_env)
	if err != nil {
		log.Fatalf("Failed to decode private key: %v\n", err)
	}

	// Load the SECP256K1 private key from the decoded bytes
	pk, _ := btcec.PrivKeyFromBytes(pkBytes)
	privateKey = secp256k1.PrivKey{
		Key: pk.Serialize(),
	}

	elderAddress = CosmosPublicKeyToCosmosAddress("elder", hex.EncodeToString(privateKey.PubKey().Bytes()))

	rollId = 1
	elderRpc = "192.168.1.6:9090"
	rpcHandler(nil, nil)
	// // Setup the HTTP server, listening on port 8545
	// http.HandleFunc("/", rpcHandler)
	// log.Println("Starting server on port 8545")
	// log.Fatal(http.ListenAndServe(":8545", nil))
}
