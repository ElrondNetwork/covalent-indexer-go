package accounts

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/ElrondNetwork/covalent-indexer-go"
	"github.com/ElrondNetwork/covalent-indexer-go/process"
	"github.com/ElrondNetwork/covalent-indexer-go/process/utility"
	"github.com/ElrondNetwork/covalent-indexer-go/schema"
	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go-core/data/indexer"
	logger "github.com/ElrondNetwork/elrond-go-logger"
)

var log = logger.GetOrCreate("covalent/process/accounts")

type accountsProcessor struct {
	shardCoordinator process.ShardCoordinator
}

// NewAccountsProcessor creates a new instance of accounts processor
func NewAccountsProcessor(shardCoordinator process.ShardCoordinator) (*accountsProcessor, error) {
	if check.IfNil(shardCoordinator) {
		return nil, covalent.ErrNilShardCoordinator
	}

	return &accountsProcessor{
		shardCoordinator: shardCoordinator,
	}, nil
}

// ProcessAccounts converts accounts data to a specific structure defined by avro schema
func (ap *accountsProcessor) ProcessAccounts(
	alteredAccounts map[string]*indexer.AlteredAccount,
	processedTxs []*schema.Transaction,
	processedSCRs []*schema.SCResult,
	processedReceipts []*schema.Receipt,
) []*schema.AccountBalanceUpdate {
	addresses := ap.getAllAddresses(processedTxs, processedSCRs, processedReceipts)
	accounts := make([]*schema.AccountBalanceUpdate, 0, len(addresses))

	for address := range addresses {
		account, err := ap.processAccount(address, alteredAccounts)
		if err != nil {
			log.Warn("cannot get account address", "address", address, "error", err)
			continue
		}

		accounts = append(accounts, account)
	}

	return accounts
}

func (ap *accountsProcessor) getAllAddresses(
	processedTxs []*schema.Transaction,
	processedSCRs []*schema.SCResult,
	processedReceipts []*schema.Receipt,
) map[string]struct{} {
	addresses := make(map[string]struct{})

	for _, tx := range processedTxs {
		ap.addAddressIfInSelfShard(addresses, tx.Sender)
		ap.addAddressIfInSelfShard(addresses, tx.Receiver)
	}

	for _, scr := range processedSCRs {
		ap.addAddressIfInSelfShard(addresses, scr.Sender)
		ap.addAddressIfInSelfShard(addresses, scr.Receiver)
	}

	for _, receipt := range processedReceipts {
		ap.addAddressIfInSelfShard(addresses, receipt.Sender)
	}

	return addresses
}

func (ap *accountsProcessor) addAddressIfInSelfShard(addresses map[string]struct{}, address []byte) {
	if bytes.Equal(address, utility.MetaChainShardAddress()) {
		return
	}
	if ap.shardCoordinator.SelfId() == ap.shardCoordinator.ComputeId(address) {
		addresses[string(address)] = struct{}{}
	}
}

func (ap *accountsProcessor) processAccount(
	address string,
	alteredAccounts map[string]*indexer.AlteredAccount,
) (*schema.AccountBalanceUpdate, error) {
	if alteredAccounts == nil {
		return nil, covalent.ErrNilAlteredAccounts
	}

	acc, ok := alteredAccounts[address]
	if !ok || acc == nil {
		return nil, fmt.Errorf("%w while extracting %s from altered accounts", covalent.ErrAccountNotFound, address)
	}

	userBalance := acc.Balance
	if len(userBalance) == 0 {
		userBalance = "0"
	}

	balanceBI, ok := big.NewInt(0).SetString(userBalance, 10)
	if !ok {
		return nil, fmt.Errorf("%w for address %s with balance %s", covalent.ErrCannotCreateBigIntFromString, address, acc.Balance)
	}

	return &schema.AccountBalanceUpdate{
		Address: []byte(address),
		Balance: utility.GetBytes(balanceBI),
		Nonce:   int64(acc.Nonce),
	}, nil
}
