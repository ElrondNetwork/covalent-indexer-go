package receipts_test

import (
	"testing"

	"github.com/ElrondNetwork/covalent-indexer-go"
	"github.com/ElrondNetwork/covalent-indexer-go/process/receipts"
	"github.com/ElrondNetwork/covalent-indexer-go/process/utility"
	"github.com/ElrondNetwork/covalent-indexer-go/schema"
	"github.com/ElrondNetwork/covalent-indexer-go/testscommon"
	"github.com/ElrondNetwork/covalent-indexer-go/testscommon/mock"
	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/data"
	"github.com/ElrondNetwork/elrond-go-core/data/receipt"
	"github.com/ElrondNetwork/elrond-go-core/data/transaction"
	"github.com/stretchr/testify/require"
)

func TestNewReceiptsProcessor(t *testing.T) {
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
		_, err := receipts.NewReceiptsProcessor(currTest.args())
		require.Equal(t, currTest.expectedErr, err)
	}
}

func generateRandomReceipt() *receipt.Receipt {
	return &receipt.Receipt{
		Value:   testscommon.GenerateRandomBigInt(),
		SndAddr: testscommon.GenerateRandomBytes(),
		Data:    testscommon.GenerateRandomBytes(),
		TxHash:  testscommon.GenerateRandomBytes(),
	}
}

func TestReceiptsProcessor_ProcessReceipts_TwoReceipts_OneNormalTx_ExpectTwoProcessedReceipts(t *testing.T) {
	pubKeyConverter := &mock.PubKeyConverterStub{}
	rp, _ := receipts.NewReceiptsProcessor(pubKeyConverter)

	receipt1 := generateRandomReceipt()
	receipt2 := generateRandomReceipt()

	txPool := map[string]data.TransactionHandler{
		"hash1": receipt1,
		"hash2": receipt2,
		"hash3": &transaction.Transaction{},
	}

	timeStamp := uint64(123)
	processedReceipts := rp.ProcessReceipts(txPool, timeStamp)

	require.Len(t, processedReceipts, 2)
	requireProcessedReceiptsContains(t, processedReceipts, receipt1, "hash1", timeStamp, pubKeyConverter)
	requireProcessedReceiptsContains(t, processedReceipts, receipt2, "hash2", timeStamp, pubKeyConverter)
}

func requireProcessedReceiptsContains(
	t *testing.T,
	processedReceipts []*schema.Receipt,
	receipt *receipt.Receipt,
	receiptHash string,
	timestamp uint64,
	pubKeyConverter core.PubkeyConverter,
) {
	expectedReceipt := &schema.Receipt{
		Hash:      []byte(receiptHash),
		Value:     receipt.GetValue().Bytes(),
		Sender:    utility.EncodePubKey(pubKeyConverter, receipt.GetSndAddr()),
		Data:      receipt.GetData(),
		TxHash:    receipt.GetTxHash(),
		Timestamp: int64(timestamp),
	}

	require.Contains(t, processedReceipts, expectedReceipt)
}
