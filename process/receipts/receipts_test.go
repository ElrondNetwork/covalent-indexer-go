package receipts_test

import (
	"math/big"
	"testing"

	"github.com/ElrondNetwork/covalent-indexer-go"
	"github.com/ElrondNetwork/covalent-indexer-go/process/receipts"
	"github.com/ElrondNetwork/covalent-indexer-go/process/utility"
	"github.com/ElrondNetwork/covalent-indexer-go/schema"
	"github.com/ElrondNetwork/covalent-indexer-go/testscommon"
	"github.com/ElrondNetwork/covalent-indexer-go/testscommon/mock"
	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/data"
	"github.com/ElrondNetwork/elrond-go-core/data/indexer"
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

// TODO: fix this test as it fails randomly
func TestReceiptsProcessor_ProcessReceipts_TwoReceipts_OneNormalTx_ExpectTwoProcessedReceipts(t *testing.T) {
	rp, _ := receipts.NewReceiptsProcessor(&mock.PubKeyConverterStub{})

	receipt1 := generateRandomReceipt()
	receipt2 := generateRandomReceipt()

	txPool := map[string]data.TransactionHandlerWithGasUsedAndFee{
		"hash1": indexer.NewTransactionHandlerWithGasAndFee(receipt1, 0, big.NewInt(0)),
		"hash2": indexer.NewTransactionHandlerWithGasAndFee(receipt2, 0, big.NewInt(0)),
		"hash3": indexer.NewTransactionHandlerWithGasAndFee(&transaction.Transaction{}, 0, big.NewInt(0)),
	}

	ret := rp.ProcessReceipts(txPool, 123)

	require.Len(t, ret, 2)

	requireProcessedReceiptEqual(t, ret[0], receipt1, "hash1", 123, &mock.PubKeyConverterStub{})
	requireProcessedReceiptEqual(t, ret[1], receipt2, "hash2", 123, &mock.PubKeyConverterStub{})
}

func requireProcessedReceiptEqual(
	t *testing.T,
	processedReceipt *schema.Receipt,
	rec *receipt.Receipt,
	receiptHash string,
	timestamp uint64,
	pubKeyConverter core.PubkeyConverter) {

	require.Equal(t, []byte(receiptHash), processedReceipt.Hash)
	require.Equal(t, rec.GetValue().Bytes(), processedReceipt.Value)
	require.Equal(t, utility.EncodePubKey(pubKeyConverter, rec.GetSndAddr()), processedReceipt.Sender)
	require.Equal(t, rec.GetData(), processedReceipt.Data)
	require.Equal(t, rec.GetTxHash(), processedReceipt.TxHash)
	require.Equal(t, int64(timestamp), processedReceipt.Timestamp)
}
