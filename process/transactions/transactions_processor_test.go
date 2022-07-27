package transactions_test

import (
	"errors"
	"math/big"
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
	"github.com/ElrondNetwork/elrond-go-core/data/block"
	"github.com/ElrondNetwork/elrond-go-core/data/indexer"
	"github.com/ElrondNetwork/elrond-go-core/data/rewardTx"
	"github.com/ElrondNetwork/elrond-go-core/data/smartContractResult"
	"github.com/ElrondNetwork/elrond-go-core/data/transaction"
	"github.com/ElrondNetwork/elrond-go-core/hashing"
	"github.com/ElrondNetwork/elrond-go-core/marshal"
	"github.com/stretchr/testify/require"
)

type headerData struct {
	header     data.HeaderHandler
	headerHash []byte
}

type transactionData struct {
	tx         data.TransactionHandler
	txHash     []byte
	headerData *headerData
}

func generateRandomTx() *transaction.Transaction {
	return &transaction.Transaction{
		Nonce:       rand.Uint64(),
		Value:       testscommon.GenerateRandomBigInt(),
		RcvAddr:     testscommon.GenerateRandomBytes(),
		SndAddr:     testscommon.GenerateRandomBytes(),
		GasLimit:    rand.Uint64(),
		GasPrice:    rand.Uint64(),
		Data:        testscommon.GenerateRandomBytes(),
		Signature:   testscommon.GenerateRandomBytes(),
		SndUserName: testscommon.GenerateRandomBytes(),
		RcvUserName: testscommon.GenerateRandomBytes(),
	}
}

func generateRandomRewardTx() *rewardTx.RewardTx {
	return &rewardTx.RewardTx{
		Round:   rand.Uint64(),
		Value:   testscommon.GenerateRandomBigInt(),
		RcvAddr: testscommon.GenerateRandomBytes(),
		Epoch:   rand.Uint32(),
	}
}

func generateRandomHeaderData() *headerData {
	return &headerData{
		header:     &block.Header{Round: rand.Uint64(), TimeStamp: rand.Uint64()},
		headerHash: testscommon.GenerateRandomBytes(),
	}
}

func generateRandomTxData(headerData *headerData) *transactionData {
	return &transactionData{
		txHash:     testscommon.GenerateRandomBytes(),
		tx:         generateRandomTx(),
		headerData: headerData,
	}
}

func generateRandomRewardTxData(headerData *headerData) *transactionData {
	return &transactionData{
		txHash:     testscommon.GenerateRandomBytes(),
		tx:         generateRandomRewardTx(),
		headerData: headerData,
	}
}

func TestNewTransactionProcessor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		args        func() (core.PubkeyConverter, hashing.Hasher, marshal.Marshalizer)
		expectedErr error
	}{
		{
			args: func() (core.PubkeyConverter, hashing.Hasher, marshal.Marshalizer) {
				return nil, &mock.HasherMock{}, &mock.MarshallerStub{}
			},
			expectedErr: covalent.ErrNilPubKeyConverter,
		},
		{
			args: func() (core.PubkeyConverter, hashing.Hasher, marshal.Marshalizer) {
				return &mock.PubKeyConverterStub{}, nil, &mock.MarshallerStub{}
			},
			expectedErr: covalent.ErrNilHasher,
		},
		{
			args: func() (core.PubkeyConverter, hashing.Hasher, marshal.Marshalizer) {
				return &mock.PubKeyConverterStub{}, &mock.HasherMock{}, nil
			},
			expectedErr: covalent.ErrNilMarshaller,
		},
		{
			args: func() (core.PubkeyConverter, hashing.Hasher, marshal.Marshalizer) {
				return &mock.PubKeyConverterStub{}, &mock.HasherMock{}, &mock.MarshallerStub{}
			},
			expectedErr: nil,
		},
	}

	for _, currTest := range tests {
		_, err := transactions.NewTransactionProcessor(currTest.args())
		require.Equal(t, currTest.expectedErr, err)
	}
}

func TestTransactionProcessor_ProcessTransactions_InvalidBody_ExpectError(t *testing.T) {
	t.Parallel()

	hData := generateRandomHeaderData()
	body := data.BodyHandler(nil)

	txp, _ := transactions.NewTransactionProcessor(&mock.PubKeyConverterStub{}, &mock.HasherMock{}, &mock.MarshallerStub{})
	_, err := txp.ProcessTransactions(hData.header, hData.headerHash, body, &indexer.Pool{})

	require.Equal(t, covalent.ErrBlockBodyAssertion, err)
}

func TestTransactionProcessor_ProcessTransactions_InvalidMarshaller_ExpectZeroProcessedTxs(t *testing.T) {
	t.Parallel()

	hData := generateRandomHeaderData()
	txData1 := generateRandomTxData(hData)

	body := &block.Body{MiniBlocks: []*block.MiniBlock{
		{
			TxHashes:        [][]byte{txData1.txHash},
			ReceiverShardID: 1,
			SenderShardID:   2,
			Type:            block.TxBlock},
	},
	}

	txPool := map[string]data.TransactionHandlerWithGasUsedAndFee{
		string(txData1.txHash): indexer.NewTransactionHandlerWithGasAndFee(txData1.tx, 0, big.NewInt(0)),
	}
	pool := &indexer.Pool{
		Txs: txPool,
	}

	errMarshaller := errors.New("err marshaller")
	txp, _ := transactions.NewTransactionProcessor(
		&mock.PubKeyConverterStub{},
		&mock.HasherMock{},
		&mock.MarshallerStub{
			MarshalCalled: func(obj interface{}) ([]byte, error) {
				return nil, errMarshaller
			},
		})
	ret, err := txp.ProcessTransactions(hData.header, hData.headerHash, body, pool)

	require.Nil(t, err)
	require.Len(t, ret, 0)
}

func TestTransactionProcessor_ProcessTransactions_EmptyRelevantBlocks_ExpectZeroProcessedTxs(t *testing.T) {
	t.Parallel()

	hData := generateRandomHeaderData()

	body := &block.Body{MiniBlocks: []*block.MiniBlock{
		{
			TxHashes:        [][]byte{},
			ReceiverShardID: 1,
			SenderShardID:   2,
			Type:            block.TxBlock},
		{
			TxHashes:        [][]byte{},
			ReceiverShardID: 3,
			SenderShardID:   4,
			Type:            block.RewardsBlock},
		{
			TxHashes:        [][]byte{},
			ReceiverShardID: 3,
			SenderShardID:   4,
			Type:            block.InvalidBlock},
	},
	}

	txp, _ := transactions.NewTransactionProcessor(&mock.PubKeyConverterStub{}, &mock.HasherMock{}, &mock.MarshallerStub{})
	ret, err := txp.ProcessTransactions(hData.header, hData.headerHash, body, &indexer.Pool{})

	require.Nil(t, err)
	require.Len(t, ret, 0)
}

func TestTransactionProcessor_ProcessTransactions_ThreeBlocks_TxsNotFoundInPool_ExpectZeroProcessedTxs(t *testing.T) {
	t.Parallel()

	hData := generateRandomHeaderData()

	body := &block.Body{MiniBlocks: []*block.MiniBlock{
		{
			TxHashes:        [][]byte{[]byte("tx not found")},
			ReceiverShardID: 1,
			SenderShardID:   2,
			Type:            block.TxBlock},
		{
			TxHashes:        [][]byte{[]byte("tx not found")},
			ReceiverShardID: 1,
			SenderShardID:   2,
			Type:            block.RewardsBlock},
		{
			TxHashes:        [][]byte{[]byte("tx not found")},
			ReceiverShardID: 1,
			SenderShardID:   2,
			Type:            block.InvalidBlock},
	},
	}

	txp, _ := transactions.NewTransactionProcessor(&mock.PubKeyConverterStub{}, &mock.HasherMock{}, &mock.MarshallerStub{})
	ret, err := txp.ProcessTransactions(hData.header, hData.headerHash, body, &indexer.Pool{})

	require.Nil(t, err)
	require.Len(t, ret, 0)
}

func TestTransactionProcessor_ProcessTransactions_OneTxBlock_OneTx_ExpectOneProcessedTx(t *testing.T) {
	t.Parallel()

	hData := generateRandomHeaderData()
	txData1 := generateRandomTxData(hData)

	body := &block.Body{MiniBlocks: []*block.MiniBlock{
		{
			TxHashes:        [][]byte{txData1.txHash},
			ReceiverShardID: 1,
			SenderShardID:   2,
			Type:            block.TxBlock},
	},
	}

	txPool := map[string]data.TransactionHandlerWithGasUsedAndFee{
		string(txData1.txHash): indexer.NewTransactionHandlerWithGasAndFee(txData1.tx, 0, big.NewInt(0)),
	}
	pool := &indexer.Pool{
		Txs: txPool,
	}

	txp, _ := transactions.NewTransactionProcessor(&mock.PubKeyConverterStub{}, &mock.HasherMock{}, &mock.MarshallerStub{})
	ret, _ := txp.ProcessTransactions(hData.header, hData.headerHash, body, pool)

	require.Len(t, ret, 1)
	requireProcessedTransactionEqual(t, ret[0], txData1, body.GetMiniBlocks()[0], &mock.PubKeyConverterStub{}, &mock.HasherMock{}, &mock.MarshallerStub{})
}

func TestTransactionProcessor_ProcessTransactions_OneRewardBlock_OneRewardTx_ExpectOneProcessedTx(t *testing.T) {
	t.Parallel()

	hData := generateRandomHeaderData()
	rewardTxData := generateRandomRewardTxData(hData)

	body := &block.Body{MiniBlocks: []*block.MiniBlock{
		{
			TxHashes:        [][]byte{rewardTxData.txHash},
			ReceiverShardID: 1,
			SenderShardID:   2,
			Type:            block.RewardsBlock},
	},
	}

	rewardsPool := map[string]data.TransactionHandlerWithGasUsedAndFee{
		string(rewardTxData.txHash): indexer.NewTransactionHandlerWithGasAndFee(rewardTxData.tx, 0, big.NewInt(0)),
	}
	pool := &indexer.Pool{
		Rewards: rewardsPool,
	}

	txp, _ := transactions.NewTransactionProcessor(&mock.PubKeyConverterStub{}, &mock.HasherMock{}, &mock.MarshallerStub{})
	ret, _ := txp.ProcessTransactions(hData.header, hData.headerHash, body, pool)

	require.Len(t, ret, 1)
	requireProcessedTransactionEqual(t, ret[0], rewardTxData, body.GetMiniBlocks()[0], &mock.PubKeyConverterStub{}, &mock.HasherMock{}, &mock.MarshallerStub{})
}

func TestTransactionProcessor_ProcessTransactions_OneInvalidBlock_OneTx_ExpectOneProcessedTx(t *testing.T) {
	t.Parallel()

	hData := generateRandomHeaderData()
	txData1 := generateRandomTxData(hData)

	body := &block.Body{MiniBlocks: []*block.MiniBlock{
		{
			TxHashes:        [][]byte{txData1.txHash},
			ReceiverShardID: 1,
			SenderShardID:   2,
			Type:            block.InvalidBlock},
	},
	}

	invalidTxPool := map[string]data.TransactionHandlerWithGasUsedAndFee{
		string(txData1.txHash): indexer.NewTransactionHandlerWithGasAndFee(txData1.tx, 0, big.NewInt(0)),
	}
	pool := &indexer.Pool{
		Invalid: invalidTxPool,
	}

	txp, _ := transactions.NewTransactionProcessor(&mock.PubKeyConverterStub{}, &mock.HasherMock{}, &mock.MarshallerStub{})
	ret, _ := txp.ProcessTransactions(hData.header, hData.headerHash, body, pool)

	require.Len(t, ret, 1)
	requireProcessedTransactionEqual(t, ret[0], txData1, body.GetMiniBlocks()[0], &mock.PubKeyConverterStub{}, &mock.HasherMock{}, &mock.MarshallerStub{})
}

func TestTransactionProcessor_ProcessTransactions_ThreeRelevantBlocks_ThreeRelevantTxs_ExpectTwoProcessedTx(t *testing.T) {
	t.Parallel()

	hData := generateRandomHeaderData()
	rewardTxData := generateRandomRewardTxData(hData)
	normalTxData := generateRandomTxData(hData)
	invalidTxData := generateRandomTxData(hData)

	body := &block.Body{MiniBlocks: []*block.MiniBlock{
		{
			TxHashes:        [][]byte{normalTxData.txHash},
			ReceiverShardID: 1,
			SenderShardID:   2,
			Type:            block.TxBlock},
		{
			TxHashes:        [][]byte{rewardTxData.txHash},
			ReceiverShardID: 3,
			SenderShardID:   4,
			Type:            block.RewardsBlock},
		{
			TxHashes:        [][]byte{invalidTxData.txHash},
			ReceiverShardID: 3,
			SenderShardID:   4,
			Type:            block.InvalidBlock},
	},
	}

	txPool := map[string]data.TransactionHandlerWithGasUsedAndFee{
		string(normalTxData.txHash): indexer.NewTransactionHandlerWithGasAndFee(normalTxData.tx, 0, big.NewInt(0)),
	}
	rewardsPool := map[string]data.TransactionHandlerWithGasUsedAndFee{
		string(rewardTxData.txHash): indexer.NewTransactionHandlerWithGasAndFee(rewardTxData.tx, 0, big.NewInt(0)),
	}
	invalidTxPool := map[string]data.TransactionHandlerWithGasUsedAndFee{
		string(invalidTxData.txHash): indexer.NewTransactionHandlerWithGasAndFee(invalidTxData.tx, 0, big.NewInt(0)),
	}
	pool := &indexer.Pool{
		Txs:     txPool,
		Rewards: rewardsPool,
		Invalid: invalidTxPool,
	}

	txp, _ := transactions.NewTransactionProcessor(&mock.PubKeyConverterStub{}, &mock.HasherMock{}, &mock.MarshallerStub{})
	ret, _ := txp.ProcessTransactions(hData.header, hData.headerHash, body, pool)

	require.Len(t, ret, 3)
	requireProcessedTransactionEqual(t, ret[0], normalTxData, body.GetMiniBlocks()[0], &mock.PubKeyConverterStub{}, &mock.HasherMock{}, &mock.MarshallerStub{})
	requireProcessedTransactionEqual(t, ret[1], rewardTxData, body.GetMiniBlocks()[1], &mock.PubKeyConverterStub{}, &mock.HasherMock{}, &mock.MarshallerStub{})
	requireProcessedTransactionEqual(t, ret[2], invalidTxData, body.GetMiniBlocks()[2], &mock.PubKeyConverterStub{}, &mock.HasherMock{}, &mock.MarshallerStub{})
}

func TestTransactionProcessor_ProcessTransactions_OneTxBLock_TwoNormalTxs_ExpectTwoProcessedTxs(t *testing.T) {
	t.Parallel()

	hData := generateRandomHeaderData()

	txData1 := generateRandomTxData(hData)
	txData2 := generateRandomTxData(hData)

	body := &block.Body{MiniBlocks: []*block.MiniBlock{
		{
			TxHashes:        [][]byte{txData1.txHash, txData2.txHash},
			ReceiverShardID: 1,
			SenderShardID:   2,
			Type:            block.TxBlock},
	},
	}

	txPool := map[string]data.TransactionHandlerWithGasUsedAndFee{
		string(txData1.txHash): indexer.NewTransactionHandlerWithGasAndFee(txData1.tx, 0, big.NewInt(0)),
		string(txData2.txHash): indexer.NewTransactionHandlerWithGasAndFee(txData2.tx, 0, big.NewInt(0)),
	}
	pool := &indexer.Pool{
		Txs: txPool,
	}

	txp, _ := transactions.NewTransactionProcessor(&mock.PubKeyConverterStub{}, &mock.HasherMock{}, &mock.MarshallerStub{})
	ret, _ := txp.ProcessTransactions(hData.header, hData.headerHash, body, pool)

	require.Len(t, ret, 2)

	requireProcessedTransactionEqual(t, ret[0], txData1, body.GetMiniBlocks()[0], &mock.PubKeyConverterStub{}, &mock.HasherMock{}, &mock.MarshallerStub{})
	requireProcessedTransactionEqual(t, ret[1], txData2, body.GetMiniBlocks()[0], &mock.PubKeyConverterStub{}, &mock.HasherMock{}, &mock.MarshallerStub{})
}

func TestTransactionProcessor_ProcessTransactions_TwoTxBlocks_TwoTxs_ExpectTwoProcessedTx(t *testing.T) {
	t.Parallel()

	hData := generateRandomHeaderData()

	txData1 := generateRandomTxData(hData)
	txData2 := generateRandomTxData(hData)

	body := &block.Body{MiniBlocks: []*block.MiniBlock{
		{
			TxHashes:        [][]byte{txData1.txHash},
			ReceiverShardID: 1,
			SenderShardID:   2,
			Type:            block.TxBlock},
		{
			TxHashes:        [][]byte{txData2.txHash},
			ReceiverShardID: 3,
			SenderShardID:   4,
			Type:            block.TxBlock},
	},
	}

	txPool := map[string]data.TransactionHandlerWithGasUsedAndFee{
		string(txData1.txHash): indexer.NewTransactionHandlerWithGasAndFee(txData1.tx, 0, big.NewInt(0)),
		string(txData2.txHash): indexer.NewTransactionHandlerWithGasAndFee(txData2.tx, 0, big.NewInt(0)),
	}
	pool := &indexer.Pool{
		Txs: txPool,
	}

	txp, _ := transactions.NewTransactionProcessor(&mock.PubKeyConverterStub{}, &mock.HasherMock{}, &mock.MarshallerStub{})
	ret, _ := txp.ProcessTransactions(hData.header, hData.headerHash, body, pool)

	require.Len(t, ret, 2)

	requireProcessedTransactionEqual(t, ret[0], txData1, body.GetMiniBlocks()[0], &mock.PubKeyConverterStub{}, &mock.HasherMock{}, &mock.MarshallerStub{})
	requireProcessedTransactionEqual(t, ret[1], txData2, body.GetMiniBlocks()[1], &mock.PubKeyConverterStub{}, &mock.HasherMock{}, &mock.MarshallerStub{})
}

func TestTransactionProcessor_ProcessTransactions_TwoRewardsBlocks_TwoRewardTxs_OneNormalTx_ExpectTwoProcessedTx(t *testing.T) {
	t.Parallel()

	hData := generateRandomHeaderData()

	normalTxData := generateRandomTxData(hData)
	rewardTxData1 := generateRandomRewardTxData(hData)
	rewardTxData2 := generateRandomRewardTxData(hData)

	body := &block.Body{MiniBlocks: []*block.MiniBlock{
		{
			TxHashes:        [][]byte{rewardTxData1.txHash, normalTxData.txHash},
			ReceiverShardID: 1,
			SenderShardID:   2,
			Type:            block.RewardsBlock},
		{
			TxHashes:        [][]byte{rewardTxData2.txHash},
			ReceiverShardID: 3,
			SenderShardID:   4,
			Type:            block.RewardsBlock},
	},
	}

	rewardsTxPool := map[string]data.TransactionHandlerWithGasUsedAndFee{
		string(rewardTxData1.txHash): indexer.NewTransactionHandlerWithGasAndFee(rewardTxData1.tx, 0, big.NewInt(0)),
		string(rewardTxData2.txHash): indexer.NewTransactionHandlerWithGasAndFee(rewardTxData2.tx, 0, big.NewInt(0)),
		string(normalTxData.txHash):  indexer.NewTransactionHandlerWithGasAndFee(normalTxData.tx, 0, big.NewInt(0)),
	}
	pool := &indexer.Pool{
		Rewards: rewardsTxPool,
	}

	txp, _ := transactions.NewTransactionProcessor(&mock.PubKeyConverterStub{}, &mock.HasherMock{}, &mock.MarshallerStub{})
	ret, _ := txp.ProcessTransactions(hData.header, hData.headerHash, body, pool)

	require.Len(t, ret, 2)

	requireProcessedTransactionEqual(t, ret[0], rewardTxData1, body.GetMiniBlocks()[0], &mock.PubKeyConverterStub{}, &mock.HasherMock{}, &mock.MarshallerStub{})
	requireProcessedTransactionEqual(t, ret[1], rewardTxData2, body.GetMiniBlocks()[1], &mock.PubKeyConverterStub{}, &mock.HasherMock{}, &mock.MarshallerStub{})
}

func TestTransactionProcessor_ProcessTransactions_OneTxBlock_OneSCRTx_ExpectZeroProcessedTxs(t *testing.T) {
	t.Parallel()

	hData := generateRandomHeaderData()
	scrHash := []byte("scr tx hash")

	body := &block.Body{MiniBlocks: []*block.MiniBlock{
		{
			TxHashes:        [][]byte{scrHash},
			ReceiverShardID: 1,
			SenderShardID:   2,
			Type:            block.TxBlock},
	},
	}

	txPool := map[string]data.TransactionHandlerWithGasUsedAndFee{
		string(scrHash): indexer.NewTransactionHandlerWithGasAndFee(&smartContractResult.SmartContractResult{}, 0, big.NewInt(0)),
	}
	pool := &indexer.Pool{
		Txs: txPool,
	}
	txp, _ := transactions.NewTransactionProcessor(&mock.PubKeyConverterStub{}, &mock.HasherMock{}, &mock.MarshallerStub{})
	ret, err := txp.ProcessTransactions(hData.header, hData.headerHash, body, pool)

	require.Nil(t, err)
	require.Len(t, ret, 0)
}

func TestTransactionProcessor_ProcessTransactions_OneSCRBlock_OneSCRTx_ExpectZeroProcessedTxs(t *testing.T) {
	t.Parallel()

	hData := generateRandomHeaderData()
	scrHash := []byte("scr tx hash")
	body := &block.Body{MiniBlocks: []*block.MiniBlock{
		{
			TxHashes:        [][]byte{scrHash},
			ReceiverShardID: 1,
			SenderShardID:   2,
			Type:            block.SmartContractResultBlock},
	},
	}

	txPool := map[string]data.TransactionHandlerWithGasUsedAndFee{
		string(scrHash): indexer.NewTransactionHandlerWithGasAndFee(&smartContractResult.SmartContractResult{}, 0, big.NewInt(0)),
	}
	pool := &indexer.Pool{
		Txs: txPool,
	}

	txp, _ := transactions.NewTransactionProcessor(&mock.PubKeyConverterStub{}, &mock.HasherMock{}, &mock.MarshallerStub{})
	ret, err := txp.ProcessTransactions(hData.header, hData.headerHash, body, pool)

	require.Nil(t, err)
	require.Len(t, ret, 0)
}

func requireNormalTxEqual(
	t *testing.T,
	processedTx *schema.Transaction,
	td *transactionData,
	pubKeyConverter core.PubkeyConverter,
) {
	tx := td.tx.(*transaction.Transaction)
	hData := td.headerData

	require.Equal(t, int64(tx.GetNonce()), processedTx.Nonce)
	require.Equal(t, utility.EncodePubKey(pubKeyConverter, tx.GetSndAddr()), processedTx.Sender)
	require.Equal(t, int64(tx.GetGasPrice()), processedTx.GasPrice)
	require.Equal(t, int64(tx.GetGasLimit()), processedTx.GasLimit)
	require.Equal(t, tx.GetData(), processedTx.Data)
	require.Equal(t, tx.GetSignature(), processedTx.Signature)
	require.Equal(t, tx.GetSndUserName(), processedTx.SenderUserName)
	require.Equal(t, tx.GetRcvUserName(), processedTx.ReceiverUserName)
	require.Equal(t, int64(hData.header.GetRound()), processedTx.Round)
}

func requireRewardTxEqual(
	t *testing.T,
	processedTx *schema.Transaction,
	tx *rewardTx.RewardTx) {
	require.Equal(t, int64(0), processedTx.Nonce)
	require.Equal(t, utility.MetaChainShardAddress(), processedTx.Sender)
	require.Equal(t, int64(0), processedTx.GasPrice)
	require.Equal(t, int64(0), processedTx.GasLimit)
	require.Equal(t, []byte(nil), processedTx.Data)
	require.Equal(t, []byte(nil), processedTx.Signature)
	require.Equal(t, []byte(nil), processedTx.SenderUserName)
	require.Equal(t, []byte(nil), processedTx.ReceiverUserName)
	require.Equal(t, int64(tx.GetRound()), processedTx.Round)
}

func requireProcessedTransactionEqual(
	t *testing.T,
	processedTx *schema.Transaction,
	td *transactionData,
	miniBlock *block.MiniBlock,
	pubKeyConverter core.PubkeyConverter,
	hasher hashing.Hasher,
	marshaller marshal.Marshalizer) {
	mbHash, _ := core.CalculateHash(marshaller, hasher, miniBlock)

	require.Equal(t, td.txHash, processedTx.Hash)
	require.Equal(t, td.tx.GetValue().Bytes(), processedTx.Value)
	require.Equal(t, utility.EncodePubKey(pubKeyConverter, td.tx.GetRcvAddr()), processedTx.Receiver)
	require.Equal(t, int32(miniBlock.GetReceiverShardID()), processedTx.ReceiverShard)
	require.Equal(t, int32(miniBlock.GetSenderShardID()), processedTx.SenderShard)
	require.Equal(t, mbHash, processedTx.MiniBlockHash)
	require.Equal(t, td.headerData.headerHash, processedTx.BlockHash)
	require.Equal(t, int64(td.headerData.header.GetTimeStamp()), processedTx.Timestamp)

	_, isNormalTx := td.tx.(*transaction.Transaction)
	if isNormalTx {
		requireNormalTxEqual(t, processedTx, td, pubKeyConverter)
	}
	rewardTransaction, isRewardTx := td.tx.(*rewardTx.RewardTx)
	if isRewardTx {
		requireRewardTxEqual(t, processedTx, rewardTransaction)
	}

}
