package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

var RollAppNonceMap = make(map[string]uint64)

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
func GetRollAppId(rollAppRPC string) (uint64, error) {
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
		return 0, err
	}

	// Forward the request to the target RPC endpoint
	resp, err := QueryRollAppRPC(rollAppRPC, requestBody)
	if err != nil {
		log.Fatalf("Failed to fetch chain ID: %v", err)
		return 0, err
	}

	// Convert the response to JSON
	jsonResp, err := RollAppRpcBytesToJsonResp(resp)
	if err != nil {
		log.Fatalf("Failed to parse JSON response: %v", err)
		return 0, err
	}

	if jsonResp.Error != nil {
		log.Fatalf("Error in RPC call: %v", jsonResp.Error)
		return 0, jsonResp.Error.(error)
	}

	result := jsonResp.Result.(string)
	rollid, err := strconv.ParseUint(result[2:], 16, 64)
	if err != nil {
		log.Fatalf("Failed to parse roll ID: %v", jsonResp.Result)
		return 0, err
	}

	return rollid, nil
}

func GetAddressNonce(rollAppRPC string, address string) (uint64, error) {
	// JSON-RPC request payload to get the transaction count (nonce)
	rpcRequest := JsonRPCRequest{
		JsonRPC: "2.0",
		Method:  "eth_getTransactionCount",
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

	if jsonResp.Error != nil {
		log.Fatalf("Error in RPC call: %v", jsonResp.Error)
		return 0, jsonResp.Error.(error)
	}

	result := jsonResp.Result.(string)
	nonce, err := strconv.ParseUint(result[2:], 16, 64)
	if err != nil {
		log.Fatalf("Failed to parse roll ID: %v", jsonResp.Result)
		return 0, err
	}

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

	chainId := tx.ChainId()                         // Get the sender address from the transaction
	chainIdFromRpc, err := GetRollAppId(rollAppRpc) // Get the chain ID from the target RPC endpoint
	if err != nil {
		log.Fatalf("Failed to fetch the chain ID: %v", err)
		return err
	}

	if chainId.Uint64() != chainIdFromRpc {
		log.Fatalf("Chain ID mismatch: %d != %d", chainId, chainIdFromRpc)
		return fmt.Errorf("Chain ID mismatch: %d != %d", chainId, chainIdFromRpc)
	}

	fromAddress, err := types.LatestSignerForChainID(chainId).Sender(&tx) // Get the sender address from the transaction
	if err != nil {
		log.Fatalf("Failed to extract the sender address: %v", err)
		return err
	}

	pubKey := privateKey.PubKey().Address()
	if fromAddress.Cmp(common.Address(pubKey)) == 0 {
		log.Fatalf("Sender address mismatch: %s != %s", fromAddress, pubKey.String())
		return fmt.Errorf("Sender address mismatch: %s != %s", fromAddress, pubKey.String())
	}

	nonce := tx.Nonce()
	var addressNonceFromRpc uint64

	if nonceFromMap, ok := RollAppNonceMap[fromAddress.String()]; ok {
		addressNonceFromRpc = nonceFromMap
	} else {
		addressNonceFromRpc, err := GetAddressNonce(rollAppRpc, fromAddress.String())
		if err != nil {
			log.Fatalf("Failed to fetch the nonce: %v", err)
			return fmt.Errorf("Failed to fetch the nonce: %v", err)
		}
		RollAppNonceMap[fromAddress.String()] = addressNonceFromRpc + uint64(1)
	}

	if nonce != addressNonceFromRpc {
		log.Fatalf("Nonce mismatch: %d != %d", nonce, addressNonceFromRpc)
		return fmt.Errorf("Nonce mismatch: %d != %d", nonce, addressNonceFromRpc)
	}

	return nil
}
