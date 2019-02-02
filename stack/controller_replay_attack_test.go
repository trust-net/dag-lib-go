// Copyright 2018-2019 The trust-net Authors
package stack

import (
	"github.com/trust-net/dag-lib-go/log"
	"github.com/trust-net/dag-lib-go/stack/dto"
	"github.com/trust-net/dag-lib-go/stack/p2p"
	"testing"
)

func TestValidateSignature_ValidPayload(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, _, _, _, _ := initMocksAndDb()
	// now switch the p2p Layer in DLT stack to real validation
	if p2pLayer, err := p2p.NewDEVp2pLayer(p2p.TestConfig(), stack.runner); err == nil {
		stack.p2p = p2pLayer
	} else {
		t.Errorf("Failed to instantiate real p2p layer: %s", err)
	}

	// create a transaction with correct submitter sequence as "nonce" value
	if _, err := stack.Submit(dto.TestSubmitter().NewRequest("test")); err != nil {
		t.Errorf("Failed to submit transaction: %s", err)
	}
}

// test payload replay attack during transaction signature valiation
func TestValidateSignature_PayloadReplayInjection(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, _, _, _, _ := initMocksAndDb()
	// now switch the p2p Layer in DLT stack to real validation
	if p2pLayer, err := p2p.NewDEVp2pLayer(p2p.TestConfig(), stack.runner); err == nil {
		stack.p2p = p2pLayer
	} else {
		t.Errorf("Failed to instantiate real p2p layer: %s", err)
	}
	// submit a valid transaction with original submitter
	submitter := dto.TestSubmitter()
	var origTx dto.Transaction
	var err error
	if origTx, err = stack.Submit(submitter.NewRequest("test")); err != nil {
		t.Errorf("Failed to submit transaction: %s", err)
	} else {
		submitter.LastTx = origTx.Id()
		submitter.Seq += 1
	}

	log.SetLogLevel(log.DEBUG)
	defer log.SetLogLevel(log.NONE)

	// now create a new transaction on a dishonest remote host that uses same request from original transaction for replay attack
	remote, _, _, _, _ := initMocksAndDb()
	replayTx, _ := remote.Submit(dto.TestSubmitter().NewRequest("random data"))
	*replayTx.Request() = *origTx.Request()

	// build a mock peer
	mockConn := p2p.TestConn()
	peer := NewMockPeer(mockConn)

	// setup mock connection to send the replay transaction followed by clean shutdown
	mockConn.NextMsg(TransactionMsgCode, replayTx)
	mockConn.NextMsg(NodeShutdownMsgCode, &NodeShutdown{})

	// setup a test event listener
	events := make(chan controllerEvent, 10)
	finished := checkForEventCode(RECV_NewTxBlockMsg, events)

	// now call stack's listener
	if err := stack.listener(peer, events); err == nil {
		t.Errorf("Transaction processing did not detect replay attack in validation")
	}
	// wait for event listener to process
	result := <-finished

	// listener should not have generated RECV_NewTxBlockMsg
	if result.seenMsgEvent {
		t.Errorf("Event listener should not generate RECV_NewTxBlockMsg event!!!")
	}
}
