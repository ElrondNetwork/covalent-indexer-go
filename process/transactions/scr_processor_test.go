package transactions_test

import (
	"math/rand"
	"testing"

	"github.com/ElrondNetwork/covalent-indexer-go"
	"github.com/ElrondNetwork/covalent-indexer-go/process/transactions"
	"github.com/ElrondNetwork/covalent-indexer-go/process/utility"
	"github.com/ElrondNetwork/covalent-indexer-go/schema"
	"github.com/ElrondNetwork/covalent-indexer-go/testscommon"
	"github.com/ElrondNetwork/covalent-indexer-go/testscommon/mock"
	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/data"
	"github.com/ElrondNetwork/elrond-go-core/data/smartContractResult"
	"github.com/ElrondNetwork/elrond-go-core/data/vm"
	"github.com/stretchr/testify/require"
)

func TestNewSCProcessor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		args        func() core.PubkeyConverter
		expectedErr error
	}{
		{
			args: func() core.PubkeyConverter {
				return nil
			},
			expectedErr: covalent.ErrNilPubKeyConverter,
		},
		{
			args: func() core.PubkeyConverter {
				return &mock.PubKeyConverterStub{}
			},
			expectedErr: nil,
		},
	}

	for _, currTest := range tests {
		_, err := transactions.NewSCResultsProcessor(currTest.args())
		require.Equal(t, currTest.expectedErr, err)
	}
}

func generateRandomSCR() *smartContractResult.SmartContractResult {
	return &smartContractResult.SmartContractResult{
		Nonce:          rand.Uint64(),
		Value:          testscommon.GenerateRandomBigInt(),
		RcvAddr:        testscommon.GenerateRandomBytes(),
		SndAddr:        testscommon.GenerateRandomBytes(),
		RelayerAddr:    testscommon.GenerateRandomBytes(),
		RelayedValue:   testscommon.GenerateRandomBigInt(),
		Code:           testscommon.GenerateRandomBytes(),
		Data:           testscommon.GenerateRandomBytes(),
		PrevTxHash:     testscommon.GenerateRandomBytes(),
		OriginalTxHash: testscommon.GenerateRandomBytes(),
		GasLimit:       rand.Uint64(),
		GasPrice:       rand.Uint64(),
		CallType:       vm.CallType(rand.Int()),
		CodeMetadata:   testscommon.GenerateRandomBytes(),
		ReturnMessage:  testscommon.GenerateRandomBytes(),
		OriginalSender: testscommon.GenerateRandomBytes(),
	}
}

func TestScProcessor_ProcessSCs_TwoSCRs_OneNormalTx_ExpectTwoProcessedSCRs(t *testing.T) {
	pubKeyConverter := &mock.PubKeyConverterStub{}
	scp, _ := transactions.NewSCResultsProcessor(pubKeyConverter)

	tx1 := generateRandomSCR()
	tx2 := generateRandomSCR()
	tx3 := generateRandomTx()

	txPool := map[string]data.TransactionHandler{
		"hash1": tx1,
		"hash2": tx2,
		"hash3": tx3,
	}

	timeStamp := uint64(123)
	processedSCR := scp.ProcessSCRs(txPool, timeStamp)

	require.Len(t, processedSCR, 2)
	requireProcessedSCRContains(t, processedSCR, tx1, "hash1", timeStamp, pubKeyConverter)
	requireProcessedSCRContains(t, processedSCR, tx2, "hash2", timeStamp, pubKeyConverter)
}

func requireProcessedSCRContains(
	t *testing.T,
	processedSCRs []*schema.SCResult,
	scr *smartContractResult.SmartContractResult,
	hash string,
	timeStamp uint64,
	pubKeyConverter core.PubkeyConverter,
) {
	expectedSCR := &schema.SCResult{
		Hash:           []byte(hash),
		Nonce:          int64(scr.GetNonce()),
		GasLimit:       int64(scr.GetGasLimit()),
		GasPrice:       int64(scr.GetGasPrice()),
		Value:          scr.GetValue().Bytes(),
		Sender:         utility.EncodePubKey(pubKeyConverter, scr.GetSndAddr()),
		Receiver:       utility.EncodePubKey(pubKeyConverter, scr.GetRcvAddr()),
		RelayerAddr:    utility.EncodePubKey(pubKeyConverter, scr.GetRelayerAddr()),
		RelayedValue:   scr.GetRelayedValue().Bytes(),
		Code:           scr.GetCode(),
		Data:           scr.GetData(),
		PrevTxHash:     scr.GetPrevTxHash(),
		OriginalTxHash: scr.GetOriginalTxHash(),
		CallType:       int32(scr.GetCallType()),
		CodeMetadata:   scr.GetCodeMetadata(),
		ReturnMessage:  scr.GetReturnMessage(),
		Timestamp:      int64(timeStamp),
	}

	require.Contains(t, processedSCRs, expectedSCR)
}
