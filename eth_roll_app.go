package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/0xElder/op-geth/core/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

// JSON-RPC request structure
type JsonRPCRequest struct {
	JsonRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      interface{}   `json:"id"`
}

// JSON-RPC response structure
type JsonRPCResponse struct {
	JsonRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result"`
	Error   interface{} `json:"error"`
	ID      interface{} `json:"id"`
}

func ForwardToRollAppRPC(w http.ResponseWriter, targetRPC string, body []byte) {
	resp, err := QueryRollAppRPC(targetRPC, body)
	if err != nil {
		http.Error(w, "Failed to forward request", http.StatusInternalServerError)
		return
	}

	// Forward the response back to the client
	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}

// Fetches the chain ID from the target RPC endpoint
func GetRollAppId(rollAppRPC string) uint64 {
	// Set up the JSON-RPC request payload
	rpcRequest := JsonRPCRequest{
		JsonRPC: "2.0",
		Method:  "eth_chainId",
		Params:  []interface{}{}, // eth_chainId does not require any parameters
		ID:      1,
	}

	// Convert the request to JSON
	requestBody, err := json.Marshal(rpcRequest)
	if err != nil {
		log.Fatalf("Failed to marshal request: %v", err)
		return 0
	}

	// Forward the request to the target RPC endpoint
	resp, err := QueryRollAppRPC(rollAppRPC, requestBody)
	if err != nil {
		log.Fatalf("Failed to fetch chain ID: %v", err)
		return 0
	}

	// Convert the response to JSON
	jsonResp, err := RollAppRpcBytesToJsonResp(resp)
	if err != nil {
		log.Fatalf("Failed to parse JSON response: %v", err)
		return 0
	}

	if rollid, ok := jsonResp.Result.(uint64); !ok {
		log.Fatalf("Failed to parse roll ID: %v", jsonResp.Result)
		return 0
	} else {
		return rollid
	}
}

func getAddressNonce(rollAppRPC string, address string) (int64, error) {
	// JSON-RPC request payload to get the transaction count (nonce)
	rpcRequest := JsonRPCRequest{
		JsonRPC: "2.0",
		Method:  "eth_chainId",
		Params:  []interface{}{address, "latest"},
		ID:      1,
	}

	// Convert the request to JSON
	requestBody, err := json.Marshal(rpcRequest)
	if err != nil {
		log.Fatalf("Failed to marshal request: %v", err)
		return 0, err
	}

	// Send the HTTP POST request
	resp, err := QueryRollAppRPC(rollAppRPC, requestBody)
	if err != nil {
		log.Fatalf("Failed to send HTTP request: %v", err)
		return 0, err
	}

	// Convert the response to JSON
	jsonResp, err := RollAppRpcBytesToJsonResp(resp)
	if err != nil {
		log.Fatalf("Failed to parse JSON response: %v", err)
		return 0, nil
	}

	// Read the response
	result := jsonResp.Result.(map[string]interface{})

	// Extract the nonce (transaction count) from the response
	if result["error"] != nil {
		log.Fatalf("Error in RPC call: %v", result["error"])
		return 0, result["error"].(error)
	}
	nonceHex := result["result"].(string)

	// Convert hex string to integer
	var nonce int64
	fmt.Sscanf(nonceHex, "0x%x", &nonce)

	return nonce, nil
}

// Function to forward the RPC request to the target endpoint
func QueryRollAppRPC(targetRPC string, body []byte) ([]byte, error) {
	// Forward the request to the target RPC endpoint
	resp, err := http.Post(targetRPC, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return []byte{}, err
	}
	defer resp.Body.Close()

	// Read the response from the target RPC
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, err
	}

	return responseBody, nil
}

func RollAppRpcBytesToJsonResp(body []byte) (JsonRPCResponse, error) {
	// Unmarshal the response into our struct
	var rpcResponse JsonRPCResponse
	err := json.Unmarshal(body, &rpcResponse)
	if err != nil {
		return JsonRPCResponse{}, err
	}

	return rpcResponse, nil
}

func VerifyReceivedRollAppTx(rollAppRpc, rawTx string) error {
	// Decode the raw transaction from hex
	txBytes, err := hex.DecodeString(rawTx)
	if err != nil {
		log.Fatalf("Failed to decode the transaction: %v", err)
		return err
	}

	// Unmarshal the transaction
	var tx types.Transaction
	err = rlp.DecodeBytes(txBytes, &tx)
	if err != nil {
		log.Fatalf("Failed to unmarshal the transaction: %v", err)
		return err
	}

	chainId := tx.ChainId()                    // Get the sender address from the transaction
	chainIdFromRpc := GetRollAppId(rollAppRpc) // Get the chain ID from the target RPC endpoint
	if chainId.Uint64() != chainIdFromRpc {
		log.Fatalf("Chain ID mismatch: %d != %d", chainId, chainIdFromRpc)
		return fmt.Errorf("Chain ID mismatch: %d != %d", chainId, chainIdFromRpc)
	}

	fromAddress, err := types.LatestSignerForChainID(chainId).Sender(&tx) // Get the sender address from the transaction
	if err != nil {
		log.Fatalf("Failed to extract the sender address: %v", err)
		return err
	}
	if fromAddress != common.Address(privateKey.PubKey().Address()) {
		log.Fatalf("Sender address mismatch: %s != %s", fromAddress, privateKey.PubKey().Address())
		return fmt.Errorf("Sender address mismatch: %s != %s", fromAddress, privateKey.PubKey().Address())
	}

	nonce := tx.Nonce()
	addressNonceFromRpc, err := getAddressNonce(rollAppRpc, fromAddress.String())
	if err != nil {
		log.Fatalf("Failed to fetch the nonce: %v", err)
		return fmt.Errorf("Failed to fetch the nonce: %v", err)
	}

	if nonce != uint64(addressNonceFromRpc) {
		log.Fatalf("Nonce mismatch: %d != %d", nonce, addressNonceFromRpc)
		return fmt.Errorf("Nonce mismatch: %d != %d", nonce, addressNonceFromRpc)
	}

	return nil
}
