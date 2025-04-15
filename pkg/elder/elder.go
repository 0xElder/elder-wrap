package elder

import (
	"fmt"
	"log"
	"sync"

	"github.com/0xElder/elder-wrap/pkg/keystore"
	"github.com/0xElder/elder/utils"
	"github.com/0xElder/elder/x/router/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ElderClient struct {
	Conn  *grpc.ClientConn
	locks map[string]*sync.Mutex
}

func NewElderClient(endpoint string, keyStore keystore.KeyStore) (*ElderClient, error) {
	elderConn, err := grpc.NewClient(endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to elder: %v", err)
	}

	locks := make(map[string]*sync.Mutex)
	keyListByElderAddress, err := keyStore.ListByElderAddress()
	if err != nil {
		return nil, fmt.Errorf("failed to list keys by evm address: %v", err)
	}

	for elderAddress, _ := range keyListByElderAddress {
		locks[elderAddress] = &sync.Mutex{}
	}

	return &ElderClient{Conn: elderConn, locks: locks}, nil
}

func (e *ElderClient) BroadCastTxn(key *keystore.Key, msg *types.MsgSubmitRollTx) (string, error) {
	log.Printf("Locking transaction to elder with address %s with roll Id: %v", key.ElderAddress, msg.RollId)
	e.locks[key.ElderAddress].Lock()
	defer e.locks[key.ElderAddress].Unlock()
	log.Printf("Broadcasting transaction to elder with address %s with roll Id: %v", key.ElderAddress, msg.RollId)

	elderTxHash, err := utils.BuildElderTxFromMsgAndBroadcast(
		utils.AuthClient(e.Conn),
		utils.TmClient(e.Conn),
		utils.TxClient(e.Conn),
		key.PrivateKey,
		msg,
		2)
	if elderTxHash == "" || err != nil {
		return elderTxHash, fmt.Errorf("failed to broadcast transaction, elderTxHash: %v, err: %v", elderTxHash, err)
	}
	log.Printf("Unlocking transaction to elder with address %s with roll Id: %v", key.ElderAddress, msg.RollId)
	return elderTxHash, nil
}
