package process

import (
	"io"

	"github.com/ElrondNetwork/covalent-indexer-go/schema"
	"github.com/ElrondNetwork/elrond-go-core/data"
	"github.com/ElrondNetwork/elrond-go-core/data/indexer"
)

// BlockHandler defines what a block processor shall do
type BlockHandler interface {
	ProcessBlock(args *indexer.ArgsSaveBlockData) (*schema.Block, error)
}

// MiniBlockHandler defines what a mini blocks processor shall do
type MiniBlockHandler interface {
	ProcessMiniBlocks(header data.HeaderHandler, body data.BodyHandler) ([]*schema.MiniBlock, error)
}

// TransactionHandler defines what a transaction processor shall do
type TransactionHandler interface {
	ProcessTransactions(
		header data.HeaderHandler,
		headerHash []byte,
		bodyHandler data.BodyHandler,
		pool *indexer.Pool) ([]*schema.Transaction, error)
}

// SCResultsHandler defines what a smart contract processor shall do
type SCResultsHandler interface {
	ProcessSCRs(transactions map[string]data.TransactionHandlerWithGasUsedAndFee, timeStamp uint64) []*schema.SCResult
}

// ReceiptHandler defines what a receipt processor shall do
type ReceiptHandler interface {
	ProcessReceipts(receipts map[string]data.TransactionHandlerWithGasUsedAndFee, timeStamp uint64) []*schema.Receipt
}

// LogHandler defines what a log processor shall do
type LogHandler interface {
	ProcessLogs(logs []*data.LogData) []*schema.Log
}

// AccountsHandler defines what an account processor shall do
type AccountsHandler interface {
	ProcessAccounts(
		alteredAccounts map[string]*indexer.AlteredAccount,
		processedTxs []*schema.Transaction,
		processedSCRs []*schema.SCResult,
		processedReceipts []*schema.Receipt) []*schema.AccountBalanceUpdate
}

// ShardCoordinator defines what a shard coordinator shall do
type ShardCoordinator interface {
	SelfId() uint32
	ComputeId(address []byte) uint32
	IsInterfaceNil() bool
}

// WSConn defines what a websocket shall do
type WSConn interface {
	io.Closer
	ReadMessage() (messageType int, p []byte, err error)
	WriteMessage(messageType int, data []byte) error
}
