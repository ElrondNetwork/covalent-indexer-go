package factory

import (
	"github.com/ElrondNetwork/covalent-indexer-go"
	"github.com/ElrondNetwork/covalent-indexer-go/process"
	"github.com/ElrondNetwork/covalent-indexer-go/process/accounts"
	blockCovalent "github.com/ElrondNetwork/covalent-indexer-go/process/block"
	"github.com/ElrondNetwork/covalent-indexer-go/process/block/miniblocks"
	"github.com/ElrondNetwork/covalent-indexer-go/process/logs"
	"github.com/ElrondNetwork/covalent-indexer-go/process/receipts"
	"github.com/ElrondNetwork/covalent-indexer-go/process/transactions"
	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/hashing"
	"github.com/ElrondNetwork/elrond-go-core/marshal"
)

// ArgsDataProcessor holds all input dependencies required by data processor factory
// in order to create a new data handler instance of type data processor
type ArgsDataProcessor struct {
	PubKeyConvertor  core.PubkeyConverter
	Hasher           hashing.Hasher
	Marshaller       marshal.Marshalizer
	ShardCoordinator process.ShardCoordinator
}

// CreateDataProcessor creates a new data handler instance of type data processor
func CreateDataProcessor(args *ArgsDataProcessor) (covalent.DataHandler, error) {
	miniBlocksHandler, err := miniblocks.NewMiniBlocksProcessor(args.Hasher, args.Marshaller)
	if err != nil {
		return nil, err
	}

	blockHandler, err := blockCovalent.NewBlockProcessor(args.Marshaller, miniBlocksHandler)
	if err != nil {
		return nil, err
	}

	transactionsHandler, err := transactions.NewTransactionProcessor(args.PubKeyConvertor, args.Hasher, args.Marshaller)
	if err != nil {
		return nil, err
	}

	receiptsHandler, err := receipts.NewReceiptsProcessor(args.PubKeyConvertor)
	if err != nil {
		return nil, err
	}

	scResultsHandler, err := transactions.NewSCResultsProcessor(args.PubKeyConvertor)
	if err != nil {
		return nil, err
	}

	logHandler, err := logs.NewLogsProcessor(args.PubKeyConvertor)
	if err != nil {
		return nil, err
	}

	accountsHandler, err := accounts.NewAccountsProcessor(args.ShardCoordinator)
	if err != nil {
		return nil, err
	}

	return process.NewDataProcessor(
		blockHandler,
		transactionsHandler,
		scResultsHandler,
		receiptsHandler,
		logHandler,
		accountsHandler)
}
