package elder

import (
	"sync"

	"github.com/0xElder/elder-wrap/pkg/keystore"
	"github.com/0xElder/elder-wrap/pkg/logging"
	"github.com/0xElder/elder/utils"
	"github.com/0xElder/elder/x/router/types"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ElderClient struct {
	Conn   *grpc.ClientConn
	locks  map[string]*sync.Mutex
	logger logging.Logger
}

func NewElderClient(endpoint string, keyStore keystore.KeyStore, logger logging.Logger) (*ElderClient, error) {
	logger.Info(nil, "Connecting to elder gRPC endpoint", "endpoint", endpoint)
	elderConn, err := grpc.NewClient(endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Error(nil, "failed to connect to elder gRPC endpoint", "error", err)
		return nil, errors.Wrap(err, "failed to connect to elder gRPC endpoint")
	}

	locks := make(map[string]*sync.Mutex)
	keyListByElderAddress, err := keyStore.ListByElderAddress()
	if err != nil {
		logger.Error(nil, "failed to list keys by elder address", "error", err)
		return nil, errors.Wrap(err, "failed to list keys by elder address")
	}

	for elderAddress, _ := range keyListByElderAddress {
		locks[elderAddress] = &sync.Mutex{}
	}

	return &ElderClient{Conn: elderConn, locks: locks, logger: logger}, nil
}

func (e *ElderClient) BroadCastTxn(key *keystore.Key, msg *types.MsgSubmitRollTx) (string, error) {
	e.logger.Debug(nil, "Broadcasting transaction", "key", key.ElderAddress, "msg", msg)
	e.locks[key.ElderAddress].Lock()
	defer e.locks[key.ElderAddress].Unlock()

	elderTxHash, err := utils.BuildElderTxFromMsgAndBroadcast(
		utils.AuthClient(e.Conn),
		utils.TmClient(e.Conn),
		utils.TxClient(e.Conn),
		key.PrivateKey,
		msg,
		2)
	if elderTxHash == "" || err != nil {
		e.logger.Error(nil, "failed to broadcast transaction", "elderTxHash", elderTxHash, "error", err)
		return elderTxHash, errors.Wrap(err, "failed to broadcast transaction")
	}
	return elderTxHash, nil
}
