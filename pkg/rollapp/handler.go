package rollapp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
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
	w.Header().Set("Content-Type", "application/json")

	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, "Failed to read request", http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

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

		tx, key, err := r.VerifyRollAppTx(context.Background(), internalTx[2:])
		if err != nil {
			response.Error = err.Error()
			return
		}
		// Check validity of transaction fields and print safely
		txHash := tx.Hash().Hex()

		var txTo string
		if tx.To() != nil {
			txTo = tx.To().Hex()
		} else {
			txTo = "contract creation"
		}

		txValue := tx.Value().String()
		evmAddress := key.EvmAddress.Hex()
		elderAddress := key.ElderAddress

		log.Printf(`
			tx_hash: %s
			tx_to: %s
			tx_value: %s
			evm_address: %s
			elder_address: %s
		`, txHash, txTo, txValue, evmAddress, elderAddress)

		internalTxBytes, err := hexutil.Decode(internalTx)
		if err != nil {
			response.Error = err.Error()
			return
		}

		authClient := utils.AuthClient(r.elderConn)
		tmClient := utils.TmClient(r.elderConn)
		txCLient := utils.TxClient(r.elderConn)

		accNum, _, err := utils.QueryElderAccount(authClient, key.ElderAddress)
		if err != nil {
			response.Error = err.Error()
			return
		}

		msg := &types.MsgSubmitRollTx{
			RollId: r.ElderRegistationId,
			TxData: internalTxBytes,
			Sender: key.ElderAddress,
			AccNum: accNum,
		}

		elderTxHash, err := utils.BuildElderTxFromMsgAndBroadcast(authClient, tmClient, txCLient, key.PrivateKey, msg, 2)
		if elderTxHash == "" || err != nil {
			response.Error = fmt.Errorf("failed to broadcast transaction, elderTxHash: %v, err: %v", elderTxHash, err)
			return
		}

		_, rollAppBlock, err := utils.GetElderTxFromHash(txCLient, elderTxHash)
		if err != nil || rollAppBlock == "" {
			response.Error = fmt.Errorf("failed to fetch elder tx, rollAppBlock: %v, err: %v", rollAppBlock, err)
			return
		}

		response.Result = tx.Hash().String()
	} else {
		// Relay all other calls to rollApp RPC
		r.ForwardtoRollAppRPC(w, body)
	}
}
