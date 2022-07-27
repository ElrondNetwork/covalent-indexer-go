package transactions

import (
	"github.com/ElrondNetwork/covalent-indexer-go"
	"github.com/ElrondNetwork/covalent-indexer-go/process/utility"
	"github.com/ElrondNetwork/covalent-indexer-go/schema"
	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go-core/data"
	"github.com/ElrondNetwork/elrond-go-core/data/block"
	erdBlock "github.com/ElrondNetwork/elrond-go-core/data/block"
	"github.com/ElrondNetwork/elrond-go-core/data/indexer"
	"github.com/ElrondNetwork/elrond-go-core/data/rewardTx"
	"github.com/ElrondNetwork/elrond-go-core/data/transaction"
	"github.com/ElrondNetwork/elrond-go-core/hashing"
	"github.com/ElrondNetwork/elrond-go-core/marshal"
	logger "github.com/ElrondNetwork/elrond-go-logger"
)

var log = logger.GetOrCreate("covalent/process/transactions/transactionProcessor")

type transactionProcessor struct {
	hasher          hashing.Hasher
	marshaller      marshal.Marshalizer
	pubKeyConverter core.PubkeyConverter
}

// NewTransactionProcessor creates a new instance of transactions processor
func NewTransactionProcessor(
	pubKeyConverter core.PubkeyConverter,
	hasher hashing.Hasher,
	marshaller marshal.Marshalizer,
) (*transactionProcessor, error) {
	if check.IfNil(pubKeyConverter) {
		return nil, covalent.ErrNilPubKeyConverter
	}
	if check.IfNil(marshaller) {
		return nil, covalent.ErrNilMarshaller
	}
	if check.IfNil(hasher) {
		return nil, covalent.ErrNilHasher
	}

	return &transactionProcessor{
		pubKeyConverter: pubKeyConverter,
		hasher:          hasher,
		marshaller:      marshaller,
	}, nil
}

// ProcessTransactions converts transactions data to a specific structure defined by avro schema
func (txp *transactionProcessor) ProcessTransactions(
	header data.HeaderHandler,
	headerHash []byte,
	bodyHandler data.BodyHandler,
	pool *indexer.Pool,
) ([]*schema.Transaction, error) {
	body, ok := bodyHandler.(*erdBlock.Body)
	if !ok {
		return nil, covalent.ErrBlockBodyAssertion
	}

	allTxs := make([]*schema.Transaction, 0, len(pool.Txs)+len(pool.Rewards)+len(pool.Invalid))
	for _, currMiniBlock := range body.MiniBlocks {
		currPool := getRelevantTxPoolBasedOnMBType(currMiniBlock, pool)
		if currPool == nil {
			continue
		}

		txsInCurrMB, err := txp.processTxsFromMiniBlock(currPool, currMiniBlock, header, headerHash, currMiniBlock.Type)
		if err != nil {
			log.Warn("transactionProcessor.processTxsFromMiniBlock", "error", err)
			continue
		}
		allTxs = append(allTxs, txsInCurrMB...)
	}

	return allTxs, nil
}

func (txp *transactionProcessor) processTxsFromMiniBlock(
	transactions map[string]data.TransactionHandlerWithGasUsedAndFee,
	miniBlock *erdBlock.MiniBlock,
	header data.HeaderHandler,
	blockHash []byte,
	mbType block.Type,
) ([]*schema.Transaction, error) {
	miniBlockHash, err := core.CalculateHash(txp.marshaller, txp.hasher, miniBlock)
	if err != nil {
		return nil, err
	}

	txsInMiniBlock := make([]*schema.Transaction, 0, len(miniBlock.TxHashes))
	for _, txHash := range miniBlock.TxHashes {
		tx, isInPool := transactions[string(txHash)]
		if !isInPool {
			log.Warn("transactionProcessor.processTxsFromMiniBlock tx hash not found in tx pool", "hash", txHash)
			continue
		}

		processedTx := txp.processTransaction(tx.GetTxHandler(), txHash, miniBlockHash, blockHash, miniBlock, header, mbType)
		if processedTx != nil {
			txsInMiniBlock = append(txsInMiniBlock, processedTx)
		}
	}

	return txsInMiniBlock, nil
}

func (txp *transactionProcessor) processTransaction(
	tx data.TransactionHandler,
	txHash []byte,
	miniBlockHash []byte,
	blockHash []byte,
	miniBlock *erdBlock.MiniBlock,
	header data.HeaderHandler,
	mbType block.Type,
) *schema.Transaction {
	var ret *schema.Transaction

	switch mbType {
	case block.TxBlock:
		ret = txp.processNormalTransaction(tx, txHash, miniBlockHash, blockHash, miniBlock, header)
	case block.RewardsBlock:
		ret = txp.processRewardTransaction(tx, txHash, miniBlockHash, blockHash, miniBlock, header)
	case block.InvalidBlock:
		ret = txp.processNormalTransaction(tx, txHash, miniBlockHash, blockHash, miniBlock, header)
	default:
		return nil
	}

	return ret
}

func (txp *transactionProcessor) processNormalTransaction(
	normalTx data.TransactionHandler,
	txHash []byte,
	miniBlockHash []byte,
	blockHash []byte,
	miniBlock *erdBlock.MiniBlock,
	header data.HeaderHandler,
) *schema.Transaction {
	tx, castOk := normalTx.(*transaction.Transaction)
	if !castOk {
		return nil
	}

	return &schema.Transaction{
		Hash:             txHash,
		MiniBlockHash:    miniBlockHash,
		BlockHash:        blockHash,
		Nonce:            int64(tx.GetNonce()),
		Round:            int64(header.GetRound()),
		Value:            utility.GetBytes(tx.GetValue()),
		Receiver:         utility.EncodePubKey(txp.pubKeyConverter, tx.GetRcvAddr()),
		Sender:           utility.EncodePubKey(txp.pubKeyConverter, tx.GetSndAddr()),
		ReceiverShard:    int32(miniBlock.ReceiverShardID),
		SenderShard:      int32(miniBlock.SenderShardID),
		GasPrice:         int64(tx.GetGasPrice()),
		GasLimit:         int64(tx.GetGasLimit()),
		Data:             tx.GetData(),
		Signature:        tx.GetSignature(),
		Timestamp:        int64(header.GetTimeStamp()),
		SenderUserName:   tx.GetSndUserName(),
		ReceiverUserName: tx.GetRcvUserName(),
	}
}

func (txp *transactionProcessor) processRewardTransaction(
	transaction data.TransactionHandler,
	txHash []byte,
	miniBlockHash []byte,
	blockHash []byte,
	miniBlock *erdBlock.MiniBlock,
	header data.HeaderHandler,
) *schema.Transaction {
	tx, castOk := transaction.(*rewardTx.RewardTx)
	if !castOk {
		return nil
	}

	return &schema.Transaction{
		Hash:             txHash,
		MiniBlockHash:    miniBlockHash,
		BlockHash:        blockHash,
		Nonce:            0,
		Round:            int64(tx.GetRound()),
		Value:            utility.GetBytes(tx.GetValue()),
		Receiver:         utility.EncodePubKey(txp.pubKeyConverter, tx.GetRcvAddr()),
		Sender:           utility.MetaChainShardAddress(),
		ReceiverShard:    int32(miniBlock.ReceiverShardID),
		SenderShard:      int32(miniBlock.SenderShardID),
		GasPrice:         0,
		GasLimit:         0,
		Data:             nil,
		Signature:        nil,
		Timestamp:        int64(header.GetTimeStamp()),
		SenderUserName:   nil,
		ReceiverUserName: nil,
	}
}

func getRelevantTxPoolBasedOnMBType(miniBlock *erdBlock.MiniBlock, pool *indexer.Pool) map[string]data.TransactionHandlerWithGasUsedAndFee {
	var ret map[string]data.TransactionHandlerWithGasUsedAndFee

	switch miniBlock.Type {
	case block.TxBlock:
		ret = pool.Txs
	case block.RewardsBlock:
		ret = pool.Rewards
	case block.InvalidBlock:
		ret = pool.Invalid
	default:
		ret = nil
	}

	return ret
}
