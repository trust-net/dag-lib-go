// Copyright 2018-2019 The trust-net Authors
package stack

import (
	"github.com/trust-net/dag-lib-go/stack/dto"
	"github.com/trust-net/dag-lib-go/stack/p2p"
	"testing"
)

// test stack controller event listener handles ALERT_DoubleSpend correctly
// when local transaction is earlier
func TestRECV_ALERT_DoubleSpend_LocalWinner(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	local, sharder, endorser, p2pLayer, testDb := initMocksAndDb()
	// create another instance, as a remote stack
	remote, _, _, _, _ := initMocksAndDb()

	// create two double spending transaction requests
	var localTx, remoteTx dto.Transaction
	var err error
	submitter := dto.TestSubmitter()
	// submit to local first
	if localTx, err = local.Submit(submitter.NewRequest("spend $10")); err != nil {
		t.Errorf("Failed to submit local transaction: %s", err)
	}
	// add some weight to remote
	remote.Submit(dto.TestSubmitter().NewRequest("request from another submitter"))
	// now add the double spending transaction to remote, which should be later in sequence/weight
	if remoteTx, err = remote.Submit(submitter.NewRequest("spend same $10 again")); err != nil {
		t.Errorf("Failed to submit remote transaction: %s", err)
	}
	p2pLayer.Reset()
	sharder.Reset()
	endorser.Reset()
	testDb.Reset()

	// build a mock peer
	mockConn := p2p.TestConn()
	peer := NewMockPeer(mockConn)

	// start stack's event listener
	events := make(chan controllerEvent, 10)
	finished := make(chan struct{}, 2)
	go func() {
		local.peerEventsListener(peer, events)
		finished <- struct{}{}
	}()

	// now emit ALERT_DoubleSpend event with the transaction from peer that caused alert
	events <- newControllerEvent(ALERT_DoubleSpend, remoteTx)
	events <- newControllerEvent(SHUTDOWN, nil)

	// wait for event listener to finish
	<-finished

	// check if event listener correctly processed the event to handle double spending alert
	// local transaction has less weight therefore should not flush local shard

	// verify that endorser gets called to fetch submitter's known shards/tx
	if !endorser.KnownShardsTxsCalled {
		t.Errorf("Endorser did not get called for submitter/seq history")
	}

	// we should not flush the local shard
	if sharder.FlushCalled {
		t.Errorf("shard flush should not get called when local transaction wins")
	}

	// we should send the ForceShardFlush message to peer
	txBytes, _ := localTx.Serialize()
	if !peer.SendCalled {
		t.Errorf("did not send any message to peer")
	} else if peer.SendMsgCode != ForceShardFlushMsgCode {
		t.Errorf("Incorrect message code send: %d", peer.SendMsgCode)
	} else if string(peer.SendMsg.(*ForceShardFlushMsg).Bytes) != string(txBytes) {
		t.Errorf("Incorrect ForceShardFlushMsg payload: %x\nExpected: %x", peer.SendMsg.(*ForceShardFlushMsg).Bytes, txBytes)
		//	} else if string(peer.SendMsg.(*ForceShardFlushMsg).Id()) != "ForceShardFlushMsg" + string(localTx.Id()) {
		//		t.Errorf("Incorrect ForceShardFlushMsg: %x\nExpected: %x", peer.SendMsg.(*ForceShardFlushMsg).Id(), "ForceShardFlushMsg" + string(localTx.Id()))
	}

	// we should not disconnect with peer, let it initiate handshake and re-sync after flush
	if peer.DisconnectCalled {
		t.Errorf("we should not disconnect peer for double spending alert")
	}
}

// test stack controller event listener handles ALERT_DoubleSpend correctly
// when remote transaction is earlier
func TestRECV_ALERT_DoubleSpend_RemoteWinner(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	local, sharder, endorser, p2pLayer, testDb := initMocksAndDb()
	// create another instance, as a remote stack
	remote, _, _, _, _ := initMocksAndDb()

	// create two double spending transaction requests
	var remoteTx dto.Transaction
	var err error
	submitter := dto.TestSubmitter()
	// submit to remote first
	if remoteTx, err = remote.Submit(submitter.NewRequest("spend $10")); err != nil {
		t.Errorf("Failed to submit remote transaction: %s", err)
	}
	// add some weight to local stack
	local.Submit(dto.TestSubmitter().NewRequest("request from another submitter"))
	// now add the double spending transaction to local stack, which should be later in sequence/weight
	if _, err = local.Submit(submitter.NewRequest("spend same $10 again")); err != nil {
		t.Errorf("Failed to submit local transaction: %s", err)
	}
	p2pLayer.Reset()
	sharder.Reset()
	endorser.Reset()
	testDb.Reset()

	// build a mock peer
	mockConn := p2p.TestConn()
	peer := NewMockPeer(mockConn)

	// start stack's event listener
	events := make(chan controllerEvent, 10)
	finished := make(chan struct{}, 2)
	go func() {
		local.peerEventsListener(peer, events)
		finished <- struct{}{}
	}()

	// now emit ALERT_DoubleSpend event with the transaction from peer that caused alert
	events <- newControllerEvent(ALERT_DoubleSpend, remoteTx)
	events <- newControllerEvent(SHUTDOWN, nil)

	// wait for event listener to finish
	<-finished

	// check if event listener correctly processed the event to handle double spending alert
	// local transaction has less weight therefore should not flush local shard

	// verify that endorser gets called to fetch submitter's known shards/tx
	if !endorser.KnownShardsTxsCalled {
		t.Errorf("Endorser did not get called for submitter/seq history")
	}

	// we should flush the local shard
	if !sharder.FlushCalled {
		t.Errorf("shard flush did not get called when remote transaction wins")
	}

	// we should send force shard sync message to peer
	if !peer.SendCalled {
		t.Errorf("should send force shard sync message to peer")
	}

	// we should not disconnect with peer, let it initiate handshake and re-sync after flush
	if peer.DisconnectCalled {
		t.Errorf("we should not disconnect peer for double spending alert")
	}
}

// stack controller listner generates RECV_ForceShardFlushMsg event for ForceShardFlushMsg message
func TestPeerListnerGeneratesEventForForceShardFlushMsg(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, _, _, _ := initMocks()

	// build a mock peer
	mockConn := p2p.TestConn()
	peer := NewMockPeer(mockConn)

	// setup mock connection to send a ForceShardFlushMsg followed by clean shutdown
	mockConn.NextMsg(ForceShardFlushMsgCode, &ForceShardFlushMsg{})
	mockConn.NextMsg(NodeShutdownMsgCode, &NodeShutdown{})

	// setup a test event listener
	events := make(chan controllerEvent, 10)
	finished := checkForEventCode(RECV_ForceShardFlushMsg, events)

	// now call stack's listener
	if err := stack.listener(peer, events); err != nil {
		t.Errorf("Transaction processing has errors: %s", err)
	}

	// wait for event listener to process
	result := <-finished

	// check if listener generate correct event
	if !result.seenMsgEvent {
		t.Errorf("Event listener did not generate RECV_ForceShardFlushMsg event!!!")
	}
}

// test stack controller event listener handles RECV_ForceShardFlushMsg correctly
// when validation to compare with local transaction shows local was earlier
func TestRECV_ForceShardFlushMsg_LocalWasEarlier(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	local, sharder, endorser, p2pLayer, testDb := initMocksAndDb()
	// create another instance, as a remote stack
	remote, _, _, _, _ := initMocksAndDb()

	// create two double spending transaction requests
	var remoteTx dto.Transaction
	var err error
	submitter := dto.TestSubmitter()
	// submit to local first
	if _, err = local.Submit(submitter.NewRequest("spend $10")); err != nil {
		t.Errorf("Failed to submit local transaction: %s", err)
	}
	// add some weight to remote
	remote.Submit(dto.TestSubmitter().NewRequest("request from another submitter"))
	// now add the double spending transaction to remote, which should be later in sequence/weight
	if remoteTx, err = remote.Submit(submitter.NewRequest("spend same $10 again")); err != nil {
		t.Errorf("Failed to submit remote transaction: %s", err)
	}
	p2pLayer.Reset()
	sharder.Reset()
	endorser.Reset()
	testDb.Reset()

	// build a mock peer
	mockConn := p2p.TestConn()
	peer := NewMockPeer(mockConn)

	// start stack's event listener
	events := make(chan controllerEvent, 10)
	finished := make(chan struct{}, 2)
	go func() {
		local.peerEventsListener(peer, events)
		finished <- struct{}{}
	}()

	// now emit RECV_ForceShardFlushMsg event with the transaction from peer that is later
	events <- newControllerEvent(RECV_ForceShardFlushMsg, NewForceShardFlushMsg(remoteTx))
	events <- newControllerEvent(SHUTDOWN, nil)

	// wait for event listener to finish
	<-finished

	// check if event listener correctly processed the event to handle double spending alert
	// local transaction has less weight therefore should not flush local shard

	// verify that endorser gets called to fetch submitter's known shards/tx
	if !endorser.KnownShardsTxsCalled {
		t.Errorf("Endorser did not get called for submitter/seq history")
	}

	// we should not flush the local shard
	if sharder.FlushCalled {
		t.Errorf("shard flush should not get called when local transaction wins")
	}

	// we should NOT send any message to peer
	if peer.SendCalled {
		t.Errorf("should not send any message to peer")
	}

	// we should disconnect with peer, it sent invalid request
	if !peer.DisconnectCalled {
		t.Errorf("we should disconnect peer for invalid alert")
	}
}

// test stack controller event listener handles RECV_ForceShardFlushMsg correctly
// when validation to compare with local transaction shows remote was earlier
func TestRECV_ForceShardFlushMsg_RemoteWasEarlier(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	local, sharder, endorser, p2pLayer, testDb := initMocksAndDb()
	// create another instance, as a remote stack
	remote, _, _, _, _ := initMocksAndDb()

	// create two double spending transaction requests
	var remoteTx dto.Transaction
	var err error
	submitter := dto.TestSubmitter()
	// submit to remote first
	if remoteTx, err = remote.Submit(submitter.NewRequest("spend $10")); err != nil {
		t.Errorf("Failed to submit remote transaction: %s", err)
	}
	// add some weight to local stack
	local.Submit(dto.TestSubmitter().NewRequest("request from another submitter"))
	// now add the double spending transaction to local stack, which should be later in sequence/weight
	if _, err = local.Submit(submitter.NewRequest("spend same $10 again")); err != nil {
		t.Errorf("Failed to submit local transaction: %s", err)
	}
	p2pLayer.Reset()
	sharder.Reset()
	endorser.Reset()
	testDb.Reset()

	// build a mock peer
	mockConn := p2p.TestConn()
	peer := NewMockPeer(mockConn)

	// start stack's event listener
	events := make(chan controllerEvent, 10)
	finished := make(chan struct{}, 2)
	go func() {
		local.peerEventsListener(peer, events)
		finished <- struct{}{}
	}()

	// now emit RECV_ForceShardFlushMsg event with the transaction from peer that is earlier
	events <- newControllerEvent(RECV_ForceShardFlushMsg, NewForceShardFlushMsg(remoteTx))
	events <- newControllerEvent(SHUTDOWN, nil)

	// wait for event listener to finish
	<-finished

	// check if event listener correctly processed the event to handle double spending alert
	// local transaction has less weight therefore should not flush local shard

	// verify that endorser gets called to fetch submitter's known shards/tx
	if !endorser.KnownShardsTxsCalled {
		t.Errorf("Endorser did not get called for submitter/seq history")
	}

	// we should flush the local shard
	if !sharder.FlushCalled {
		t.Errorf("shard flush did not get called when remote transaction wins")
	}

	// we should send force shard sync message to peer
	if !peer.SendCalled {
		t.Errorf("should send force shard sync message to peer")
	}

	// we should not disconnect with peer, let it initiate handshake and re-sync after flush
	if peer.DisconnectCalled {
		t.Errorf("we should not disconnect peer for double spending alert")
	}
}
