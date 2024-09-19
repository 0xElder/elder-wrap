package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
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
