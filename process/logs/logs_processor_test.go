package logs_test

import (
	"testing"

	"github.com/ElrondNetwork/covalent-indexer-go"
	"github.com/ElrondNetwork/covalent-indexer-go/process/logs"
	"github.com/ElrondNetwork/covalent-indexer-go/process/utility"
	"github.com/ElrondNetwork/covalent-indexer-go/schema"
	"github.com/ElrondNetwork/covalent-indexer-go/testscommon"
	"github.com/ElrondNetwork/covalent-indexer-go/testscommon/mock"
	"github.com/ElrondNetwork/elrond-go-core/core"
	"github.com/ElrondNetwork/elrond-go-core/core/check"
	"github.com/ElrondNetwork/elrond-go-core/data/indexer"
	"github.com/ElrondNetwork/elrond-go-core/data/transaction"
	"github.com/stretchr/testify/require"
)

func TestNewLogsProcessor(t *testing.T) {
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
		_, err := logs.NewLogsProcessor(currTest.args())
		require.Equal(t, currTest.expectedErr, err)
	}
}

func TestLogsProcessor_ProcessLogs_OneNilLog_ExpectZeroProcessedLogs(t *testing.T) {
	lp, _ := logs.NewLogsProcessor(&mock.PubKeyConverterStub{})

	logsAndEvents := []indexer.LogData{
		{
			TxHash:     "hash1",
			LogHandler: nil,
		},
	}

	ret := lp.ProcessLogs(logsAndEvents)
	require.Len(t, ret, 0)

}

func TestLogsProcessor_ProcessLogs_OneLog_NoEvent_ExpectOneProcessedLogsAndZeroEvents(t *testing.T) {
	lp, _ := logs.NewLogsProcessor(&mock.PubKeyConverterStub{})

	log := &transaction.Log{
		Address: testscommon.GenerateRandomBytes(),
		Events:  []*transaction.Event{},
	}
	logsAndEvents := []indexer.LogData{
		{
			TxHash:     "hash1",
			LogHandler: log,
		},
	}

	ret := lp.ProcessLogs(logsAndEvents)

	require.Len(t, ret, 1)
	require.Len(t, ret[0].Events, 0)
}

func TestLogsProcessor_ProcessLogs_OneLog_OneEvent_ExpectOneProcessedLogAndOneEvent(t *testing.T) {
	lp, _ := logs.NewLogsProcessor(&mock.PubKeyConverterStub{})

	event := generateRandomEvent()
	log := &transaction.Log{
		Address: testscommon.GenerateRandomBytes(),
		Events:  []*transaction.Event{event},
	}

	logsAndEvents := []indexer.LogData{
		{
			TxHash:     "hash1",
			LogHandler: log,
		},
	}

	ret := lp.ProcessLogs(logsAndEvents)
	require.Len(t, ret, 1)
	require.Len(t, ret[0].Events, 1)

	requireProcessedLogEqual(t, ret[0], log, "hash1", &mock.PubKeyConverterStub{})
}

func TestLogsProcessor_ProcessLogs_ThreeLogs_FourEvents_ExpectTwoProcessedLogsAndThreeEvents(t *testing.T) {
	lp, _ := logs.NewLogsProcessor(&mock.PubKeyConverterStub{})

	event1 := generateRandomEvent()
	event2 := generateRandomEvent()
	event3 := generateRandomEvent()
	log1 := &transaction.Log{
		Address: testscommon.GenerateRandomBytes(),
		Events:  []*transaction.Event{event1, nil, event2},
	}

	log2 := &transaction.Log{
		Address: testscommon.GenerateRandomBytes(),
		Events:  []*transaction.Event{event3},
	}

	logsAndEvents := []indexer.LogData{
		{
			TxHash:     "hash1",
			LogHandler: log1,
		},
		{
			TxHash:     "hash2",
			LogHandler: nil,
		},
		{
			TxHash:     "hash3",
			LogHandler: log2,
		},
	}

	ret := lp.ProcessLogs(logsAndEvents)
	require.Len(t, ret, 2)
	require.Len(t, ret[0].Events, 2)
	require.Len(t, ret[1].Events, 1)

	requireProcessedLogEqual(t, ret[0], log1, "hash1", &mock.PubKeyConverterStub{})
	requireProcessedLogEqual(t, ret[1], log2, "hash3", &mock.PubKeyConverterStub{})
}

func generateRandomEvent() *transaction.Event {
	return &transaction.Event{
		Address:    testscommon.GenerateRandomBytes(),
		Identifier: testscommon.GenerateRandomBytes(),
		Topics:     [][]byte{testscommon.GenerateRandomBytes(), testscommon.GenerateRandomBytes()},
		Data:       testscommon.GenerateRandomBytes(),
	}
}

func requireProcessedLogEqual(
	t *testing.T,
	processedLog *schema.Log,
	log *transaction.Log,
	hash string,
	pubKeyConverter core.PubkeyConverter) {

	require.Equal(t, []byte(hash), processedLog.ID)
	require.Equal(t, utility.EncodePubKey(pubKeyConverter, log.GetAddress()), processedLog.Address)

	notNilEvents := getNotNilEvents(log.GetEvents())
	for idx, currEvent := range notNilEvents {
		requireProcessedEventEqual(t, processedLog.Events[idx], currEvent, pubKeyConverter)
	}
}

func getNotNilEvents(events []*transaction.Event) []*transaction.Event {
	ret := make([]*transaction.Event, 0, len(events))

	for _, event := range events {
		if !check.IfNil(event) {
			ret = append(ret, event)
		}
	}

	return ret
}

func requireProcessedEventEqual(
	t *testing.T,
	processedEvent *schema.Event,
	event *transaction.Event,
	pubKeyConverter core.PubkeyConverter) {

	require.Equal(t, utility.EncodePubKey(pubKeyConverter, event.GetAddress()), processedEvent.Address)
	require.Equal(t, event.GetIdentifier(), processedEvent.Identifier)
	require.Equal(t, event.GetTopics(), processedEvent.Topics)
	require.Equal(t, event.GetData(), processedEvent.Data)
}
