package accounts_test

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"testing"

	"github.com/ElrondNetwork/covalent-indexer-go"
	"github.com/ElrondNetwork/covalent-indexer-go/process"
	"github.com/ElrondNetwork/covalent-indexer-go/process/accounts"
	"github.com/ElrondNetwork/covalent-indexer-go/process/utility"
	"github.com/ElrondNetwork/covalent-indexer-go/schema"
	"github.com/ElrondNetwork/covalent-indexer-go/testscommon"
	"github.com/ElrondNetwork/covalent-indexer-go/testscommon/mock"
	"github.com/ElrondNetwork/elrond-go-core/data/indexer"
	"github.com/stretchr/testify/require"
)

func TestNewAccountsProcessor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		args        func() process.ShardCoordinator
		expectedErr error
	}{
		{
			args: func() process.ShardCoordinator {
				return nil
			},
			expectedErr: covalent.ErrNilShardCoordinator,
		},
		{
			args: func() process.ShardCoordinator {
				return &mock.ShardCoordinatorMock{}
			},
			expectedErr: nil,
		},
	}

	for _, currTest := range tests {
		_, err := accounts.NewAccountsProcessor(currTest.args())
		require.Equal(t, currTest.expectedErr, err)
	}
}

func TestAccountsProcessor_ProcessAccounts_NotInSameShard_ExpectZeroAccounts(t *testing.T) {
	ap, _ := accounts.NewAccountsProcessor(
		&mock.ShardCoordinatorMock{SelfID: 4})

	tx := &schema.Transaction{
		Receiver: testscommon.GenerateRandomBytes(),
		Sender:   testscommon.GenerateRandomBytes()}
	ret := ap.ProcessAccounts(map[string]*indexer.AlteredAccount{}, []*schema.Transaction{tx}, []*schema.SCResult{}, []*schema.Receipt{})

	require.Len(t, ret, 0)
}

func TestAccountsProcessor_ProcessAccounts_OneSender_NilReceiver_ExpectOneAccount(t *testing.T) {
	addresses := generateAddresses(1)
	ap, _ := accounts.NewAccountsProcessor(&mock.ShardCoordinatorMock{})

	tx := &schema.Transaction{
		Sender:   addresses[0],
		Receiver: nil,
	}

	alteredAccounts := map[string]*indexer.AlteredAccount{
		string(addresses[0]): {},
	}

	alteredAccounts = prepareAlteredAccounts(addresses)
	ret := ap.ProcessAccounts(alteredAccounts, []*schema.Transaction{tx}, []*schema.SCResult{}, []*schema.Receipt{})

	require.Len(t, ret, 1)
	checkProcessedAccounts(t, addresses, ret)
}

func TestAccountsProcessor_ProcessAccounts_NilSender_OneReceiver_ExpectOneAccount(t *testing.T) {
	addresses := generateAddresses(1)
	ap, _ := accounts.NewAccountsProcessor(&mock.ShardCoordinatorMock{})

	tx := &schema.Transaction{
		Sender:   nil,
		Receiver: addresses[0],
	}

	alteredAccounts := prepareAlteredAccounts(addresses)

	ret := ap.ProcessAccounts(alteredAccounts, []*schema.Transaction{tx}, []*schema.SCResult{}, []*schema.Receipt{})

	require.Len(t, ret, 1)
	checkProcessedAccounts(t, addresses, ret)
}

func TestAccountsProcessor_ProcessAccounts_FourAddresses_TwoIdentical_ExpectTwoAccounts(t *testing.T) {
	addresses := generateAddresses(2)
	ap, _ := accounts.NewAccountsProcessor(&mock.ShardCoordinatorMock{})

	tx1 := &schema.Transaction{
		Sender:   addresses[0],
		Receiver: addresses[1],
	}
	tx2 := &schema.Transaction{
		Sender:   addresses[1],
		Receiver: addresses[0],
	}

	alteredAccounts := prepareAlteredAccounts(addresses)
	ret := ap.ProcessAccounts(alteredAccounts, []*schema.Transaction{tx1, tx2}, []*schema.SCResult{}, []*schema.Receipt{})

	require.Len(t, ret, 2)
	checkProcessedAccounts(t, addresses, ret)
}

func TestAccountsProcessor_ProcessAccounts_OneAddress_OneMetaChainShardAddress_ExpectOneAccount(t *testing.T) {
	ap, _ := accounts.NewAccountsProcessor(&mock.ShardCoordinatorMock{})

	addresses := generateAddresses(1)
	tx := &schema.Transaction{
		Receiver: []byte("adr0"),
		Sender:   utility.MetaChainShardAddress(),
	}

	alteredAccounts := prepareAlteredAccounts(addresses)
	ret := ap.ProcessAccounts(alteredAccounts, []*schema.Transaction{tx}, []*schema.SCResult{}, []*schema.Receipt{})

	require.Len(t, ret, 1)
	checkProcessedAccounts(t, addresses, ret)
}

func TestAccountsProcessor_ProcessAccounts_SevenAddresses_ExpectSevenAccounts(t *testing.T) {
	addresses := generateAddresses(7)

	ap, _ := accounts.NewAccountsProcessor(&mock.ShardCoordinatorMock{})

	tx1 := &schema.Transaction{
		Receiver: addresses[0],
		Sender:   addresses[1],
	}
	tx2 := &schema.Transaction{
		Sender:   addresses[2],
		Receiver: addresses[3],
	}
	txs := []*schema.Transaction{tx1, tx2}

	scr := &schema.SCResult{
		Sender:   addresses[4],
		Receiver: addresses[5],
	}
	scrs := []*schema.SCResult{scr}

	receipt := &schema.Receipt{
		Sender: addresses[6],
	}
	receipts := []*schema.Receipt{receipt}

	alteredAccounts := prepareAlteredAccounts(addresses)
	ret := ap.ProcessAccounts(alteredAccounts, txs, scrs, receipts)

	require.Len(t, ret, 7)
	checkProcessedAccounts(t, addresses, ret)
}

func TestAccountsProcessor_ProcessAccounts_EmptyAlteredAccountBalance(t *testing.T) {
	t.Parallel()

	ap, _ := accounts.NewAccountsProcessor(&mock.ShardCoordinatorMock{})

	alteredAccountsMap := map[string]*indexer.AlteredAccount{
		"adr0": {
			Balance: "",
		},
	}

	tx1 := &schema.Transaction{
		Receiver: []byte("adr0"),
	}

	res := ap.ProcessAccounts(alteredAccountsMap, []*schema.Transaction{tx1}, []*schema.SCResult{}, []*schema.Receipt{})

	require.Len(t, res, 1)
	require.Equal(t, big.NewInt(0).Bytes(), res[0].Balance)
}

func generateAddresses(n int) [][]byte {
	addresses := make([][]byte, n)

	for i := 0; i < n; i++ {
		addresses[i] = []byte("adr" + strconv.Itoa(i))
	}

	return addresses
}

func prepareAlteredAccounts(addresses [][]byte) map[string]*indexer.AlteredAccount {
	mapToRet := make(map[string]*indexer.AlteredAccount)

	for idx, addr := range addresses {
		mapToRet[string(addr)] = &indexer.AlteredAccount{
			Balance: big.NewInt(0).SetInt64(int64(idx)).String(),
			Nonce:   big.NewInt(0).SetInt64(int64(idx)).Uint64(),
		}
	}

	return mapToRet
}

// This function relies on the order of the addresses according to generateAddresses(). for example, adr1 needs to have
// a balance of 1 and a nonce of 1
func checkProcessedAccounts(t *testing.T, addresses [][]byte, processedAcc []*schema.AccountBalanceUpdate) {
	require.Equal(t, len(addresses), len(processedAcc), "should have the same number of processed accounts as initial addresses")

	allProcessedAddr := make(map[string]struct{})

	for _, currAccount := range processedAcc {
		allProcessedAddr[string(currAccount.Address)] = struct{}{}
	}

	for _, addr := range addresses {
		_, exists := allProcessedAddr[string(addr)]
		require.True(t, exists, fmt.Sprintf("%s not processed successfully", addr))
	}

	for _, account := range processedAcc {
		addrIdxStr := strings.Split(string(account.Address), "adr")[1] //adr0 => "0"
		addrIdxBI, _ := big.NewInt(0).SetString(addrIdxStr, 10)
		addrIdx := addrIdxBI.Int64()
		require.Equal(t, big.NewInt(addrIdx).Bytes(), account.Balance)
		require.Equal(t, addrIdx, account.Nonce)
	}
}
