package covalent

import (
	"bytes"
	"encoding/hex"
	"sync"
	"time"

	"github.com/ElrondNetwork/covalent-indexer-go/process"
	"github.com/ElrondNetwork/covalent-indexer-go/process/utility"
	"github.com/ElrondNetwork/elrond-go-core/data"
	"github.com/ElrondNetwork/elrond-go-core/data/indexer"
	logger "github.com/ElrondNetwork/elrond-go-logger"
	"github.com/gorilla/websocket"
)

var log = logger.GetOrCreate("covalent")

const RetrialTimeoutMS = 50

type covalentIndexer struct {
	processor        DataHandler
	server           HttpServer
	wss              process.WSConn
	mutWSS           sync.RWMutex
	wsr              process.WSConn
	mutWSR           sync.RWMutex
	newConnectionWSR chan struct{}
	newConnectionWSS chan struct{}
}

// NewCovalentDataIndexer creates a new instance of covalent data indexer, which implements Driver interface and
// converts protocol input data to covalent required data
func NewCovalentDataIndexer(processor DataHandler, server HttpServer) (*covalentIndexer, error) {
	if processor == nil {
		return nil, ErrNilDataHandler
	}
	if server == nil {
		return nil, ErrNilHTTPServer
	}
	ci := &covalentIndexer{
		processor: processor,
		server:    server,
	}
	ci.newConnectionWSR = make(chan struct{})
	ci.newConnectionWSS = make(chan struct{})

	go ci.start()

	return ci, nil
}

func (ci *covalentIndexer) SetWSSender(wss process.WSConn) {
	ci.mutWSS.Lock()
	if ci.wss != nil {
		err := ci.wss.Close()
		log.LogIfError(err)
	}
	ci.wss = wss
	ci.mutWSS.Unlock()

	ci.newConnectionWSS <- struct{}{}
}

func (ci *covalentIndexer) SetWSReceiver(wsr process.WSConn) {
	ci.mutWSR.Lock()
	if ci.wsr != nil {
		err := ci.wsr.Close()
		log.LogIfError(err)
	}
	ci.wsr = wsr
	ci.mutWSR.Unlock()

	ci.newConnectionWSR <- struct{}{}
}

func (ci *covalentIndexer) getWSS() process.WSConn {
	ci.mutWSS.RLock()
	defer ci.mutWSS.RUnlock()

	return ci.wss
}

func (ci *covalentIndexer) getWSR() process.WSConn {
	ci.mutWSR.RLock()
	defer ci.mutWSR.RUnlock()

	return ci.wsr
}

func (ci *covalentIndexer) waitForWSSConnection() {
	for {
		select {
		case <-ci.newConnectionWSS:
			return
		}
	}
}

func (ci *covalentIndexer) waitForWSRConnection() {
	for {
		select {
		case <-ci.newConnectionWSR:
			return
		}
	}
}

func (ci *covalentIndexer) start() {
	err := ci.server.ListenAndServe()
	if err != nil {
		log.Error("could not initialize webserver", "error", err)
	}
}

func (ci *covalentIndexer) sendWithRetrial(data []byte, ackData []byte) {
	wss := ci.getWSS()
	wsr := ci.getWSR()

	if wss == nil {
		ci.waitForWSSConnection()
	}
	if wsr == nil {
		ci.waitForWSRConnection()
	}

	ticker := time.NewTicker(time.Millisecond * RetrialTimeoutMS)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			wss = ci.getWSS()
			wsr = ci.getWSR()

			if wss != nil && wsr != nil {
				dataSent := ci.sendDataWithAcknowledge(data, ackData, wss, wsr)
				if dataSent {
					return
				}
			}
		}
	}
}

func (ci *covalentIndexer) sendDataWithAcknowledge(
	data []byte,
	ackData []byte,
	wss process.WSConn,
	wsr process.WSConn,
) bool {
	errSend := wss.WriteMessage(websocket.BinaryMessage, data)
	if errSend != nil {
		log.Warn("could not send block data to covalent, waiting for new connection", "error", errSend)
		ci.waitForWSSConnection()
	}

	msgType, receivedData, errReadData := wsr.ReadMessage()
	if errReadData != nil {
		log.Warn("could not receive acknowledge data from covalent, waiting for new connection", "error", errReadData)
		ci.waitForWSRConnection()
	}

	if errSend == nil && errReadData == nil && msgType == websocket.BinaryMessage {
		if bytes.Equal(receivedData, ackData) {
			return true
		}
	}

	return false
}

// SaveBlock saves the block info and converts it in order to be sent to covalent
func (ci *covalentIndexer) SaveBlock(args *indexer.ArgsSaveBlockData) error {
	blockResult, err := ci.processor.ProcessData(args)
	if err != nil {
		log.Error("SaveBlock failed. Could not process block",
			"error", err, "headerHash", hex.EncodeToString(args.HeaderHash))
		panic("could not process block, check log")
	}

	dataToSend, err := utility.Encode(blockResult)
	if err != nil {
		log.Error("could not encode block result to binary data", "error", err)
		panic("could not encode block result, check log")
	}

	ci.sendWithRetrial(dataToSend, blockResult.Block.Hash)

	return nil
}

// RevertIndexedBlock returns nil
func (ci *covalentIndexer) RevertIndexedBlock(data.HeaderHandler, data.BodyHandler) error {
	return nil
}

// SaveRoundsInfo returns nil
func (ci *covalentIndexer) SaveRoundsInfo(_ []*indexer.RoundInfo) error {
	return nil
}

// SaveValidatorsPubKeys returns nil
func (ci *covalentIndexer) SaveValidatorsPubKeys(map[uint32][][]byte, uint32) error {
	return nil
}

// SaveValidatorsRating returns nil
func (ci *covalentIndexer) SaveValidatorsRating(string, []*indexer.ValidatorRatingInfo) error {
	return nil
}

// SaveAccounts returns nil
func (ci *covalentIndexer) SaveAccounts(uint64, []data.UserAccountHandler) error {
	return nil
}

// FinalizedBlock returns nil
func (ci *covalentIndexer) FinalizedBlock(_ []byte) error {
	return nil
}

// Close closes websocket connections(if they exist) as well as the server which listens for new connections
func (ci *covalentIndexer) Close() error {
	wss := ci.getWSS()
	wsr := ci.getWSR()

	if wss != nil {
		err := wss.Close()
		log.LogIfError(err)
	}
	if wsr != nil {
		err := wsr.Close()
		log.LogIfError(err)
	}

	if ci.server != nil {
		return ci.server.Close()
	}
	return nil
}

// IsInterfaceNil returns true if there is no value under the interface
func (ci *covalentIndexer) IsInterfaceNil() bool {
	return ci == nil
}
