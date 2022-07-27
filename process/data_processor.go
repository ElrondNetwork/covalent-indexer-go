package process

import (
	"github.com/ElrondNetwork/covalent-indexer-go/schema"
	"github.com/ElrondNetwork/elrond-go-core/data"
	"github.com/ElrondNetwork/elrond-go-core/data/indexer"
)

type dataProcessor struct {
	blockHandler       BlockHandler
	transactionHandler TransactionHandler
	receiptHandler     ReceiptHandler
	scHandler          SCResultsHandler
	logHandler         LogHandler
	accountsHandler    AccountsHandler
}

// NewDataProcessor creates a new instance of data processor, which handles all sub-processes
func NewDataProcessor(
	blockHandler BlockHandler,
	transactionHandler TransactionHandler,
	scHandler SCResultsHandler,
	receiptHandler ReceiptHandler,
	logHandler LogHandler,
	accountsHandler AccountsHandler,
) (*dataProcessor, error) {

	return &dataProcessor{
		blockHandler:       blockHandler,
		transactionHandler: transactionHandler,
		scHandler:          scHandler,
		receiptHandler:     receiptHandler,
		logHandler:         logHandler,
		accountsHandler:    accountsHandler,
	}, nil
}

// ProcessData converts all covalent necessary data to a specific structure defined by avro schema
func (dp *dataProcessor) ProcessData(args *indexer.ArgsSaveBlockData) (*schema.BlockResult, error) {
	pool := getPool(args)

	block, err := dp.blockHandler.ProcessBlock(args)
	if err != nil {
		return nil, err
	}

	transactions, err := dp.transactionHandler.ProcessTransactions(args.Header, args.HeaderHash, args.Body, pool)
	if err != nil {
		return nil, err
	}

	smartContractResults := dp.scHandler.ProcessSCRs(pool.Scrs, args.Header.GetTimeStamp())
	receipts := dp.receiptHandler.ProcessReceipts(pool.Receipts, args.Header.GetTimeStamp())
	logs := dp.logHandler.ProcessLogs(pool.Logs)
	accountUpdates := dp.accountsHandler.ProcessAccounts(args.AlteredAccounts, transactions, smartContractResults, receipts)

	return &schema.BlockResult{
		Block:        block,
		Transactions: transactions,
		Receipts:     receipts,
		SCResults:    smartContractResults,
		Logs:         logs,
		StateChanges: accountUpdates,
	}, nil
}

func getPool(args *indexer.ArgsSaveBlockData) *indexer.Pool {
	pool := &indexer.Pool{
		Txs:      make(map[string]data.TransactionHandlerWithGasUsedAndFee),
		Scrs:     make(map[string]data.TransactionHandlerWithGasUsedAndFee),
		Rewards:  make(map[string]data.TransactionHandlerWithGasUsedAndFee),
		Invalid:  make(map[string]data.TransactionHandlerWithGasUsedAndFee),
		Receipts: make(map[string]data.TransactionHandlerWithGasUsedAndFee),
		Logs:     make([]*data.LogData, 0),
	}
	if args.TransactionsPool != nil {
		pool = args.TransactionsPool
	}

	return pool
}
