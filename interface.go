package covalent

import (
	"github.com/ElrondNetwork/covalent-indexer-go/schema"
	"github.com/ElrondNetwork/elrond-go-core/data"
	"github.com/ElrondNetwork/elrond-go-core/data/indexer"
	vmcommon "github.com/ElrondNetwork/elrond-vm-common"
)

type DataHandler interface {
	ProcessData(args *indexer.ArgsSaveBlockData) (*schema.BlockResult, error)
}

type Driver interface {
	SaveBlock(args *indexer.ArgsSaveBlockData) error
	RevertIndexedBlock(header data.HeaderHandler, body data.BodyHandler) error
	SaveRoundsInfo(roundsInfos []*indexer.RoundInfo) error
	SaveValidatorsPubKeys(validatorsPubKeys map[uint32][][]byte, epoch uint32) error
	SaveValidatorsRating(indexID string, infoRating []*indexer.ValidatorRatingInfo) error
	SaveAccounts(blockTimestamp uint64, acc []data.UserAccountHandler) error
	FinalizedBlock(headerHash []byte) error
	Close() error
	IsInterfaceNil() bool
}

type AccountsAdapter interface {
	LoadAccount(address []byte) (vmcommon.AccountHandler, error)
	IsInterfaceNil() bool
}

type HttpServer interface {
	ListenAndServe() error
	Close() error
}
