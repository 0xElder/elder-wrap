package rollapp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/0xElder/elder/utils"
	"github.com/0xElder/elder/x/router/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type JsonRPCRequest struct {
	JsonRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      interface{}   `json:"id"`
}

// JSON-RPC response structure
type JsonRPCResponse struct {
	JsonRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
	ID      interface{} `json:"id"`
}

func (r *RollApp) HandleRequest(w http.ResponseWriter, req *http.Request) {
	logger := r.logger.With("method", "HandleRequest")
	w.Header().Set("Content-Type", "application/json")

	body, err := io.ReadAll(req.Body)
	if err != nil {
		logger.Error(req.Context(), "Failed to read request body", "error", err)
		http.Error(w, "Failed to read request", http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	// Handle batch requests
	if isBatch(body) {
		var rpcRequests []JsonRPCRequest
		err = json.Unmarshal(body, &rpcRequests)
		if err != nil {
			logger.Error(req.Context(), "Failed to unmarshal batch request", "error", err)
			http.Error(w, "Invalid JSON-RPC request", http.StatusBadRequest)
			return
		}

		for _, rpcRequest := range rpcRequests {
			if rpcRequest.Method == "eth_sendRawTransaction" {
				logger.Error(req.Context(), "Batch request contains eth_sendRawTransaction, not supported")
				http.Error(w, "Batch request contains eth_sendRawTransaction, not supported", http.StatusBadRequest)
				return
			}
		}

		// Relay batch requests to rollApp RPC if there are no eth_sendRawTransaction
		r.ForwardtoRollAppRPC(w, body)
		return
	}

	var rpcRequest JsonRPCRequest
	err = json.Unmarshal(body, &rpcRequest)
	if err != nil {
		logger.Error(req.Context(), "Failed to unmarshal request", "error", err)
		http.Error(w, "Invalid JSON-RPC request", http.StatusBadRequest)
		return
	}

	logger.Debug(req.Context(), "Received JSON-RPC request", "method", rpcRequest.Method, "params", rpcRequest.Params)

	// Check if the method is `eth_sendRawTransaction` (signed transaction)
	if rpcRequest.Method == "eth_sendRawTransaction" {
		logger.Debug(req.Context(), "Received eth_sendRawTransaction request")
		response := JsonRPCResponse{
			JsonRPC: rpcRequest.JsonRPC,
			ID:      rpcRequest.ID,
		}

		// Send the response back
		defer func() {
			err := json.NewEncoder(w).Encode(response)
			if err != nil {
				logger.Error(req.Context(), "Failed to encode response", "error", err)
				http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			}
		}()

		internalTx, ok := rpcRequest.Params[0].(string)
		if !ok {
			logger.Error(req.Context(), "Invalid transaction format", "params", rpcRequest.Params)
			response.Error = fmt.Errorf("invalid transaction format")
			return
		}

		if internalTx[0:2] != "0x" {
			internalTx = "0x" + internalTx
		}

		tx, key, err := r.VerifyRollAppTx(context.Background(), internalTx[2:])
		if err != nil {
			logger.Error(req.Context(), "Failed to verify transaction", "error", err)
			response.Error = err.Error()
			return
		}

		internalTxBytes, err := hexutil.Decode(internalTx)
		if err != nil {
			logger.Error(req.Context(), "Failed to decode transaction", "error", err)
			response.Error = err.Error()
			return
		}

		accNum, _, err := utils.QueryElderAccount(utils.AuthClient(r.elderClient.Conn), key.ElderAddress)
		if err != nil {
			logger.Error(req.Context(), "Failed to query elder account", "error", err)
			response.Error = err.Error()
			return
		}

		msg := &types.MsgSubmitRollTx{
			RollId: r.ElderRegistationId,
			TxData: internalTxBytes,
			Sender: key.ElderAddress,
			AccNum: accNum,
		}

		elderTxHash, err := r.elderClient.BroadCastTxn(key, msg)
		if err != nil {
			logger.Error(req.Context(), "Failed to broadcast transaction", "error", err)
			response.Error = err.Error()
			return
		}

		_, rollAppBlock, err := utils.GetElderTxFromHash(utils.TxClient(r.elderClient.Conn), elderTxHash)
		if err != nil || rollAppBlock == "" {
			logger.Error(req.Context(), "Failed to fetch elder transaction", "error", err)
			response.Error = fmt.Errorf("failed to fetch elder tx, rollAppBlock: %v, err: %v", rollAppBlock, err)
			return
		}

		response.Result = tx.Hash().String()
	} else {
		// Relay all other calls to rollApp RPC
		r.ForwardtoRollAppRPC(w, body)
	}
}

// isBatch returns true when the first non-whitespace characters is '['
// Code taken from go-ethereum/rpc/json.go
func isBatch(raw json.RawMessage) bool {
	for _, c := range raw {
		// skip insignificant whitespace (http://www.ietf.org/rfc/rfc4627.txt)
		if c == 0x20 || c == 0x09 || c == 0x0a || c == 0x0d {
			continue
		}
		return c == '['
	}
	return false
}
