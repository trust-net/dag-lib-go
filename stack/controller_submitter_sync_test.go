// Copyright 2018-2019 The trust-net Authors
package stack

import (
	"github.com/trust-net/dag-lib-go/log"
	"github.com/trust-net/dag-lib-go/stack/dto"
	"github.com/trust-net/dag-lib-go/stack/p2p"
	"testing"
)

func private() {
	// a phony method to keep log package in imports
	log.SetLogLevel(log.DEBUG)
	defer log.SetLogLevel(log.NONE)
}

// stack controller listner generates RECV_SubmitterWalkUpRequestMsg event for SubmitterWalkUpRequestMsg message
func TestPeerListnerGeneratesEventForSubmitterWalkUpRequestMsg(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, _, _, _ := initMocks()

	// build a mock peer
	mockConn := p2p.TestConn()
	peer := NewMockPeer(mockConn)

	// setup mock connection to send a SubmitterWalkUpRequestMsg followed by clean shutdown
	mockConn.NextMsg(SubmitterWalkUpRequestMsgCode, &SubmitterWalkUpRequestMsg{})
	mockConn.NextMsg(NodeShutdownMsgCode, &NodeShutdown{})

	// setup a test event listener
	events := make(chan controllerEvent, 10)
	finished := checkForEventCode(RECV_SubmitterWalkUpRequestMsg, events)

	// now call stack's listener
	if err := stack.listener(peer, events); err != nil {
		t.Errorf("Transaction processing has errors: %s", err)
	}

	// wait for event listener to process
	result := <-finished

	// check if listener generate correct event
	if !result.seenMsgEvent {
		t.Errorf("Event listener did not generate RECV_SubmitterWalkUpRequestMsg event!!!")
	}
}

// test stack controller event listener handles RECV_SubmitterWalkUpRequestMsg correctly
func TestRECV_SubmitterWalkUpRequestMsg_HasEntry(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, sharder, endorser, p2pLayer, testDb := initMocksAndDb()

	// submit a transactions to add one ancestor to local submitter history
	submitter := dto.TestSubmitter()
	tx, _ := stack.Submit(submitter.NewRequest("test payload1"))
	submitter.LastTx = tx.Id()
	submitter.Seq += 1

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
		stack.peerEventsListener(peer, events)
		finished <- struct{}{}
	}()

	// now emit RECV_SubmitterWalkUpRequestMsg event requesting history using submitter's request from transaction
	req := NewSubmitterWalkUpRequestMsg(submitter.NewRequest("dummy"))
	events <- newControllerEvent(RECV_SubmitterWalkUpRequestMsg, req)
	events <- newControllerEvent(SHUTDOWN, nil)

	// wait for event listener to finish
	<-finished

	// check if event listener correctly processed the event to handle submitter's walk up request

	// verify that endorser gets called to fetch submitter's known shards/tx
	if !endorser.KnownShardsTxsCalled {
		t.Errorf("Endorser did not get called for submitter/seq history")
	}

	// we should have sent the SubmitterWalkUpResponseMsg back with correct transaction/shard from submitter/seq history
	if !peer.SendCalled {
		t.Errorf("did not send any message to peer")
	} else if peer.SendMsgCode != SubmitterWalkUpResponseMsgCode {
		t.Errorf("Incorrect message code send: %d", peer.SendMsgCode)
	} else if string(peer.SendMsg.(*SubmitterWalkUpResponseMsg).Submitter) != string(req.Submitter) {
		t.Errorf("Incorrect response submitter: %x\nExpected: %x", peer.SendMsg.(*SubmitterWalkUpResponseMsg).Submitter, req.Submitter)
	} else if peer.SendMsg.(*SubmitterWalkUpResponseMsg).Seq != req.Seq {
		t.Errorf("Incorrect response sequence: %x\nExpected: %x", peer.SendMsg.(*SubmitterWalkUpResponseMsg).Seq, req.Seq)
	} else if len(peer.SendMsg.(*SubmitterWalkUpResponseMsg).Transactions) != 1 {
		t.Errorf("Incorrect number of transactions: %d", len(peer.SendMsg.(*SubmitterWalkUpResponseMsg).Transactions))
	} else if len(peer.SendMsg.(*SubmitterWalkUpResponseMsg).Shards) != 1 {
		t.Errorf("Incorrect number of shards: %d", len(peer.SendMsg.(*SubmitterWalkUpResponseMsg).Shards))
	} else if peer.SendMsg.(*SubmitterWalkUpResponseMsg).Transactions[0] != tx.Id() {
		t.Errorf("Incorrect transaction in response: %x\nExpected: %x", peer.SendMsg.(*SubmitterWalkUpResponseMsg).Transactions[0], tx.Id())
	} else if string(peer.SendMsg.(*SubmitterWalkUpResponseMsg).Shards[0]) != string(tx.Request().ShardId) {
		t.Errorf("Incorrect shard in response: %x\nExpected: %x", peer.SendMsg.(*SubmitterWalkUpResponseMsg).Shards[0], tx.Request().ShardId)
	}
}

// stack controller listner generates RECV_SubmitterWalkUpResponseMsg event for SubmitterWalkUpResponseMsg message
func TestPeerListnerGeneratesEventForSubmitterWalkUpResponseMsg(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, _, _, _ := initMocks()

	// build a mock peer
	mockConn := p2p.TestConn()
	peer := NewMockPeer(mockConn)

	// setup mock connection to send a SubmitterWalkUpRequestMsg followed by clean shutdown
	mockConn.NextMsg(SubmitterWalkUpResponseMsgCode, &SubmitterWalkUpResponseMsg{})
	mockConn.NextMsg(NodeShutdownMsgCode, &NodeShutdown{})

	// setup a test event listener
	events := make(chan controllerEvent, 10)
	finished := checkForEventCode(RECV_SubmitterWalkUpResponseMsg, events)

	// now call stack's listener
	if err := stack.listener(peer, events); err != nil {
		t.Errorf("Transaction processing has errors: %s", err)
	}

	// wait for event listener to process
	result := <-finished

	// check if listener generate correct event
	if !result.seenMsgEvent {
		t.Errorf("Event listener did not generate RECV_SubmitterWalkUpResponseMsg event!!!")
	}
}

// test stack controller event listener handles RECV_SubmitterWalkUpResponseMsg correctly for unknown shard
func TestRECV_SubmitterWalkUpResponseMsg_UnknownShard(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, sharder, endorser, p2pLayer, testDb := initMocksAndDb()

	// submit a transactions to add ancestor to local shard's Anchor
	tx, _ := stack.Submit(dto.TestSubmitter().NewRequest("test payload1"))
	// build a SubmitterWalkUpResponseMsg that has same submitter, seq but different shard
	msg := NewSubmitterWalkUpResponseMsg(NewSubmitterWalkUpRequestMsg(tx.Request()))
	msg.Transactions = [][64]byte{dto.RandomHash()}
	msg.Shards = [][]byte{[]byte("a different shard")}
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
		stack.peerEventsListener(peer, events)
		finished <- struct{}{}
	}()

	// now emit RECV_SubmitterWalkUpResponseMsg event providing history with an unknown shard
	events <- newControllerEvent(RECV_SubmitterWalkUpResponseMsg, msg)
	events <- newControllerEvent(SHUTDOWN, nil)

	// wait for event listener to finish
	<-finished

	// check if event listener correctly processed the event to handle submitter's walk up request

	// verify that endorser gets called to fetch submitter's known shards/tx
	if !endorser.KnownShardsTxsCalled {
		t.Errorf("Endorser did not get called for submitter/seq history")
	}

	// we should have continued walk up because of unknow shard
	// we should have sent the submitter sync message
	if !peer.SendCalled {
		t.Errorf("did not send submitter sync message to peer")
	} else if peer.SendMsgCode != SubmitterWalkUpRequestMsgCode {
		t.Errorf("Incorrect message code send: %d", peer.SendMsgCode)
	} else if string(peer.SendMsg.(*SubmitterWalkUpRequestMsg).Submitter) != string(msg.Submitter) {
		t.Errorf("Incorrect SubmitterWalkUpRequestMsg Submitter: %s", peer.SendMsg.(*SubmitterWalkUpRequestMsg).Submitter)
	} else if peer.SendMsg.(*SubmitterWalkUpRequestMsg).Seq != msg.Seq-1 {
		t.Errorf("Incorrect SubmitterWalkUpRequestMsg Sequence: %d", peer.SendMsg.(*SubmitterWalkUpRequestMsg).Seq)
	}
}

// test stack controller event listener handles RECV_SubmitterWalkUpResponseMsg correctly for double spend
func TestRECV_SubmitterWalkUpResponseMsg_DoubleSpend(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, sharder, endorser, p2pLayer, testDb := initMocksAndDb()

	// submit a transactions to add ancestor to local shard's Anchor
	tx, _ := stack.Submit(dto.TestSubmitter().NewRequest("test payload1"))

	// build a SubmitterWalkUpResponseMsg that has same submitter, seq, shard but different transaction
	msg := NewSubmitterWalkUpResponseMsg(NewSubmitterWalkUpRequestMsg(tx.Request()))
	msg.Transactions = [][64]byte{dto.RandomHash()}
	msg.Shards = [][]byte{tx.Request().ShardId}
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
		stack.peerEventsListener(peer, events)
		finished <- struct{}{}
	}()

	// now emit RECV_SubmitterWalkUpResponseMsg event providing history with a double spend transaction
	events <- newControllerEvent(RECV_SubmitterWalkUpResponseMsg, msg)
	events <- newControllerEvent(SHUTDOWN, nil)

	// wait for event listener to finish
	<-finished

	// check if event listener correctly processed the event to handle submitter's walk up request

	// verify that endorser gets called to fetch submitter's known shards/tx
	if !endorser.KnownShardsTxsCalled {
		t.Errorf("Endorser did not get called for submitter/seq history")
	}

	// we should NOT have initiate a double spend resolution, and instead just keep walking
	// up the history till we find a sequence which matches locally
	if len(events) != 0 {
		t.Errorf("should not emit ALERT_DoubleSpend")
	}
	if !peer.SendCalled {
		t.Errorf("did not send submitter sync message to peer")
	} else if peer.SendMsgCode != SubmitterWalkUpRequestMsgCode {
		t.Errorf("Incorrect message code send: %d", peer.SendMsgCode)
	} else if string(peer.SendMsg.(*SubmitterWalkUpRequestMsg).Submitter) != string(msg.Submitter) {
		t.Errorf("Incorrect SubmitterWalkUpRequestMsg Submitter: %s", peer.SendMsg.(*SubmitterWalkUpRequestMsg).Submitter)
	} else if peer.SendMsg.(*SubmitterWalkUpRequestMsg).Seq != msg.Seq-1 {
		t.Errorf("Incorrect SubmitterWalkUpRequestMsg Sequence: %d", peer.SendMsg.(*SubmitterWalkUpRequestMsg).Seq)
	}
}

// test stack controller event listener handles RECV_SubmitterWalkUpResponseMsg correctly when atleast one transaction is unknown
func TestRECV_SubmitterWalkUpResponseMsg_ContinueWalkUp(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, sharder, endorser, p2pLayer, testDb := initMocksAndDb()

	// setup DLT DB with submitter history using 1 shard/transaction pair for the submitter/seq
	submitter := []byte("a test submitter")
	seq := uint64(0x21)
	tx1 := TestSignedTransaction("test payload1")
	tx1.Request().SubmitterId = submitter
	tx1.Request().SubmitterSeq = seq
	tx1.Request().ShardId = []byte("shard 1")
	if err := testDb.UpdateSubmitter(tx1); err != nil {
		t.Errorf("Failed to update submitter history for transaction 1: %s", err)
	}
	// build another shard/transaction pair that is NOT in local DLT DB
	tx2 := TestSignedTransaction("test payload2")
	tx2.Request().SubmitterId = submitter
	tx2.Request().SubmitterSeq = seq
	tx2.Request().ShardId = []byte("shard 2")

	// build a SubmitterWalkUpResponseMsg that has above submitter/seq and both shard/transaction pairs
	msg := &SubmitterWalkUpResponseMsg{
		Submitter: submitter,
		Seq:       seq,
	}
	msg.Transactions = [][64]byte{tx1.Id(), tx2.Id()}
	msg.Shards = [][]byte{tx1.Request().ShardId, tx2.Request().ShardId}
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
		stack.peerEventsListener(peer, events)
		finished <- struct{}{}
	}()

	// now emit RECV_SubmitterWalkUpResponseMsg event providing history with one known and one unknown pair
	events <- newControllerEvent(RECV_SubmitterWalkUpResponseMsg, msg)
	events <- newControllerEvent(SHUTDOWN, nil)

	// wait for event listener to finish
	<-finished

	// check if event listener correctly processed the event to handle submitter's walk up request

	// verify that endorser gets called to fetch submitter's known shards/tx
	if !endorser.KnownShardsTxsCalled {
		t.Errorf("Endorser did not get called for submitter/seq history")
	}

	// we should have continued walk up because of 1 unknown pair
	// we should have sent the submitter sync message
	if !peer.SendCalled {
		t.Errorf("did not send submitter sync message to peer")
	} else if peer.SendMsgCode != SubmitterWalkUpRequestMsgCode {
		t.Errorf("Incorrect message code send: %d", peer.SendMsgCode)
	} else if string(peer.SendMsg.(*SubmitterWalkUpRequestMsg).Submitter) != string(msg.Submitter) {
		t.Errorf("Incorrect SubmitterWalkUpRequestMsg Submitter: %s", peer.SendMsg.(*SubmitterWalkUpRequestMsg).Submitter)
	} else if peer.SendMsg.(*SubmitterWalkUpRequestMsg).Seq != msg.Seq-1 {
		t.Errorf("Incorrect SubmitterWalkUpRequestMsg Sequence: %d", peer.SendMsg.(*SubmitterWalkUpRequestMsg).Seq)
	}
}

// test stack controller event listener handles RECV_SubmitterWalkUpResponseMsg correctly for all known shards/transactions
func TestRECV_SubmitterWalkUpResponseMsg_AllKnownPairs(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, sharder, endorser, p2pLayer, testDb := initMocksAndDb()

	// submit a transactions to add ancestor to local shard's Anchor
	sub := dto.TestSubmitter()
	tx, _ := stack.Submit(sub.NewRequest("test payload1"))
	sub.LastTx = tx.Id()
	sub.Seq += 1

	// build a SubmitterWalkUpResponseMsg that has same submitter, seq, shard and transaction
	msg := NewSubmitterWalkUpResponseMsg(NewSubmitterWalkUpRequestMsg(sub.NewRequest("dummy")))
	msg.Transactions = [][64]byte{tx.Id()}
	msg.Shards = [][]byte{tx.Request().ShardId}
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
		stack.peerEventsListener(peer, events)
		finished <- struct{}{}
	}()

	// now emit RECV_SubmitterWalkUpResponseMsg event providing history with known local history
	events <- newControllerEvent(RECV_SubmitterWalkUpResponseMsg, msg)
	events <- newControllerEvent(SHUTDOWN, nil)

	// wait for event listener to finish
	<-finished

	// check if event listener correctly processed the event to handle submitter's walk up request

	// verify that endorser gets called to fetch submitter's known shards/tx
	if !endorser.KnownShardsTxsCalled {
		t.Errorf("Endorser did not get called for submitter/seq history")
	}

	// we should have transitioned to ProcessDownStage
	// we should have sent the SubmitterProcessDownRequestMsg message to fetch submitter's history for next sequence from peer
	if !peer.SendCalled {
		t.Errorf("did not send any message to peer")
	} else if peer.SendMsgCode != SubmitterProcessDownRequestMsgCode {
		t.Errorf("Incorrect message code send: %d", peer.SendMsgCode)
	} else if string(peer.SendMsg.(*SubmitterProcessDownRequestMsg).Submitter) != string(msg.Submitter) {
		t.Errorf("Incorrect submitter: %x\nExpected: %x", peer.SendMsg.(*SubmitterProcessDownRequestMsg).Submitter, msg.Submitter)
	} else if peer.SendMsg.(*SubmitterProcessDownRequestMsg).Seq != msg.Seq+1 {
		t.Errorf("Incorrect sequence: %d\nExpected: %d", peer.SendMsg.(*SubmitterProcessDownRequestMsg).Seq, msg.Seq+1)
	}
}

// test stack controller event listener handles RECV_SubmitterWalkUpResponseMsg correctly when no pairs in response for non 0 seq
func TestRECV_SubmitterWalkUpResponseMsg_ZeroPairsNonZeroSeq(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, sharder, endorser, p2pLayer, testDb := initMocksAndDb()

	// setup DLT DB with submitter history using 1 shard/transaction pair for the submitter/seq
	submitter := []byte("a test submitter")
	seq := uint64(0x21)
	tx1 := TestSignedTransaction("test payload1")
	tx1.Request().SubmitterId = submitter
	tx1.Request().SubmitterSeq = seq
	tx1.Request().ShardId = []byte("shard 1")
	if err := testDb.UpdateSubmitter(tx1); err != nil {
		t.Errorf("Failed to update submitter history for transaction 1: %s", err)
	}
	// build a SubmitterWalkUpResponseMsg that has above submitter/seq bot no shard/transaction pairs
	msg := &SubmitterWalkUpResponseMsg{
		Submitter: submitter,
		Seq:       tx1.Request().SubmitterSeq,
	}
	msg.Transactions = [][64]byte{}
	msg.Shards = [][]byte{}
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
		stack.peerEventsListener(peer, events)
		finished <- struct{}{}
	}()

	// now emit RECV_SubmitterWalkUpResponseMsg event providing history with one known and one unknown pair
	events <- newControllerEvent(RECV_SubmitterWalkUpResponseMsg, msg)
	events <- newControllerEvent(SHUTDOWN, nil)

	// wait for event listener to finish
	<-finished

	// check if event listener correctly processed the event to handle submitter's walk up request

	// verify that endorser did not get called to fetch submitter's known shards/tx
	if endorser.KnownShardsTxsCalled {
		t.Errorf("Endorser should not get called for invalid submitter/seq history")
	}

	// since we recieved 0 pairs for a seq that is non zero -- this is an error, peer sent us a shard/transaction for submitter
	// but has no previous history for that submitter's earlier sequences

	// we should have disconnected with peer
	if !peer.DisconnectCalled {
		t.Errorf("did not disconncect with peer")
	}
}

// test stack controller event listener handles RECV_SubmitterWalkUpResponseMsg correctly when pairs mismatch in response
func TestRECV_SubmitterWalkUpResponseMsg_MismatchShardTxCount(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, sharder, endorser, p2pLayer, testDb := initMocksAndDb()

	// setup DLT DB with submitter history using 1 shard/transaction pair for the submitter/seq
	submitter := []byte("a test submitter")
	seq := uint64(0x21)
	tx1 := TestSignedTransaction("test payload1")
	tx1.Request().SubmitterId = submitter
	tx1.Request().SubmitterSeq = seq
	tx1.Request().ShardId = []byte("shard 1")
	if err := testDb.UpdateSubmitter(tx1); err != nil {
		t.Errorf("Failed to update submitter history for transaction 1: %s", err)
	}
	// build a SubmitterWalkUpResponseMsg that has above submitter/seq but mismatch in count of shard/transaction pairs
	msg := &SubmitterWalkUpResponseMsg{
		Submitter: submitter,
		Seq:       tx1.Request().SubmitterSeq,
	}
	msg.Transactions = [][64]byte{dto.RandomHash(), dto.RandomHash(), dto.RandomHash()}
	msg.Shards = [][]byte{[]byte("shard 1"), []byte("shard 2")}
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
		stack.peerEventsListener(peer, events)
		finished <- struct{}{}
	}()

	// now emit RECV_SubmitterWalkUpResponseMsg event providing history with one known and one unknown pair
	events <- newControllerEvent(RECV_SubmitterWalkUpResponseMsg, msg)
	events <- newControllerEvent(SHUTDOWN, nil)

	// wait for event listener to finish
	<-finished

	// check if event listener correctly processed the event to handle submitter's walk up request

	// verify that endorser did not get called to fetch submitter's known shards/tx
	if endorser.KnownShardsTxsCalled {
		t.Errorf("Endorser should not get called for invalid submitter/seq history")
	}

	// we recieved not same count of shards and transactions -- this is an error, peer sent us incorrect
	// history for that submitter

	// we should have disconnected with peer
	if !peer.DisconnectCalled {
		t.Errorf("did not disconncect with peer")
	}
}

// test stack controller event listener handles RECV_SubmitterWalkUpResponseMsg correctly with peer state validation
//func TestRECV_SubmitterWalkUpResponseMsg_StateValidationFailed(t *testing.T) {
func testRECV_SubmitterWalkUpResponseMsg_StateValidationFailed(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, sharder, endorser, p2pLayer, testDb := initMocksAndDb()

	// setup DLT DB with submitter history using 1 shard/transaction pair for the submitter/seq
	submitter := []byte("a test submitter")
	seq := uint64(0x21)
	tx1 := TestSignedTransaction("test payload1")
	tx1.Request().SubmitterId = submitter
	tx1.Request().SubmitterSeq = seq
	tx1.Request().ShardId = []byte("shard 1")
	if err := testDb.UpdateSubmitter(tx1); err != nil {
		t.Errorf("Failed to update submitter history for transaction 1: %s", err)
	}
	// build a SubmitterWalkUpResponseMsg that has above submitter/seq/shard/tx
	msg := &SubmitterWalkUpResponseMsg{
		Submitter: submitter,
		Seq:       seq,
	}
	msg.Transactions = [][64]byte{tx1.Id()}
	msg.Shards = [][]byte{tx1.Request().ShardId}
	p2pLayer.Reset()
	sharder.Reset()
	endorser.Reset()
	testDb.Reset()

	// build a mock peer
	mockConn := p2p.TestConn()
	peer := NewMockPeer(mockConn)
	// set the peer state to a value different from message ID
	peer.SetState(int(RECV_SubmitterWalkUpResponseMsg), []byte("some random value"))

	// start stack's event listener
	events := make(chan controllerEvent, 10)
	finished := make(chan struct{}, 2)
	go func() {
		stack.peerEventsListener(peer, events)
		finished <- struct{}{}
	}()

	// now emit RECV_SubmitterWalkUpResponseMsg event providing correct history, but not matching peer state
	events <- newControllerEvent(RECV_SubmitterWalkUpResponseMsg, msg)
	events <- newControllerEvent(SHUTDOWN, nil)

	// wait for event listener to finish
	<-finished

	// check if event listener correctly processed the event to handle submitter's walk up request

	// verify that endorser did not get called to fetch submitter's known shards/tx
	if endorser.KnownShardsTxsCalled {
		t.Errorf("Endorser should not get called for mismatched peer state")
	}

	// we recieved a message that did not match peer state
	// we should have disconnected with peer
	if !peer.DisconnectCalled {
		t.Errorf("did not disconncect with peer")
	}
}

// stack controller listner generates RECV_SubmitterProcessDownRequestMsg event for SubmitterProcessDownRequestMsg message
func TestPeerListnerGeneratesEventForSubmitterProcessDownRequestMsg(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, _, _, _ := initMocks()

	// build a mock peer
	mockConn := p2p.TestConn()
	peer := NewMockPeer(mockConn)

	// setup mock connection to send a SubmitterProcessDownRequestMsg followed by clean shutdown
	mockConn.NextMsg(SubmitterProcessDownRequestMsgCode, &SubmitterProcessDownRequestMsg{})
	mockConn.NextMsg(NodeShutdownMsgCode, &NodeShutdown{})

	// setup a test event listener
	events := make(chan controllerEvent, 10)
	finished := checkForEventCode(RECV_SubmitterProcessDownRequestMsg, events)

	// now call stack's listener
	if err := stack.listener(peer, events); err != nil {
		t.Errorf("Transaction processing has errors: %s", err)
	}

	// wait for event listener to process
	result := <-finished

	// check if listener generate correct event
	if !result.seenMsgEvent {
		t.Errorf("Event listener did not generate RECV_SubmitterProcessDownRequestMsg event!!!")
	}
}

// test stack controller event listener handles RECV_SubmitterProcessDownRequestMsg correctly
func TestRECV_SubmitterProcessDownRequestMsg_HasEntry(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, sharder, endorser, p2pLayer, testDb := initMocksAndDb()

	// setup DLT DB with submitter history using 2 shard/transaction pairs for the submitter/seq
	submitter := []byte("a test submitter")
	seq := uint64(0x21)
	tx1 := TestSignedTransaction("test payload1")
	tx1.Request().SubmitterId = submitter
	tx1.Request().SubmitterSeq = seq
	tx1.Request().ShardId = []byte("shard 1")
	if err := testDb.UpdateSubmitter(tx1); err != nil {
		t.Errorf("Failed to update submitter history for transaction 1: %s", err)
	} else if err := testDb.UpdateShard(tx1); err != nil {
		t.Errorf("Failed to update shard DAG for transaction 1: %s", err)
	} else if err := testDb.AddTx(tx1); err != nil {
		t.Errorf("Failed to save transaction 1: %s", err)
	}
	tx2 := TestSignedTransaction("test payload2")
	tx2.Request().SubmitterId = submitter
	tx2.Request().SubmitterSeq = seq
	tx2.Request().ShardId = []byte("shard 2")
	if err := testDb.UpdateSubmitter(tx2); err != nil {
		t.Errorf("Failed to update submitter history for transaction 2: %s", err)
	} else if err := testDb.UpdateShard(tx2); err != nil {
		t.Errorf("Failed to update shard DAG for transaction 2: %s", err)
	} else if err := testDb.AddTx(tx2); err != nil {
		t.Errorf("Failed to save transaction 2: %s", err)
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
		stack.peerEventsListener(peer, events)
		finished <- struct{}{}
	}()

	// now emit RECV_SubmitterProcessDownRequestMsg event requesting history using submitter's seq setup above
	req := &SubmitterProcessDownRequestMsg{
		Submitter: submitter,
		Seq:       seq,
	}
	events <- newControllerEvent(RECV_SubmitterProcessDownRequestMsg, req)
	events <- newControllerEvent(SHUTDOWN, nil)

	// wait for event listener to finish
	<-finished

	// check if event listener correctly processed the event to handle submitter's process down request

	// verify that endorser gets called to fetch submitter's known shards/tx
	if !endorser.KnownShardsTxsCalled {
		t.Errorf("Endorser did not get called for submitter/seq history")
	}

	// we should have sent the SubmitterProcessDownResponseMsg back with correct transactions from submitter/seq history
	tx1Bytes, _ := tx1.Serialize()
	tx2Bytes, _ := tx2.Serialize()
	if !peer.SendCalled {
		t.Errorf("did not send any message to peer")
	} else if peer.SendMsgCode != SubmitterProcessDownResponseMsgCode {
		t.Errorf("Incorrect message code send: %d", peer.SendMsgCode)
	} else if string(peer.SendMsg.(*SubmitterProcessDownResponseMsg).Submitter) != string(req.Submitter) {
		t.Errorf("Incorrect response submitter: %x\nExpected: %x", peer.SendMsg.(*SubmitterProcessDownResponseMsg).Submitter, req.Submitter)
	} else if peer.SendMsg.(*SubmitterProcessDownResponseMsg).Seq != req.Seq {
		t.Errorf("Incorrect response sequence: %x\nExpected: %x", peer.SendMsg.(*SubmitterProcessDownResponseMsg).Seq, req.Seq)
	} else if len(peer.SendMsg.(*SubmitterProcessDownResponseMsg).TxBytes) != 2 {
		t.Errorf("Incorrect number of transactions: %d", len(peer.SendMsg.(*SubmitterProcessDownResponseMsg).TxBytes))
	} else if string(peer.SendMsg.(*SubmitterProcessDownResponseMsg).TxBytes[0]) != string(tx1Bytes) {
		t.Errorf("Incorrect transaction 1 in response: %x\nExpected: %x", peer.SendMsg.(*SubmitterProcessDownResponseMsg).TxBytes[0], tx1Bytes)
	} else if string(peer.SendMsg.(*SubmitterProcessDownResponseMsg).TxBytes[1]) != string(tx2Bytes) {
		t.Errorf("Incorrect transaction 2 in response: %x\nExpected: %x", peer.SendMsg.(*SubmitterProcessDownResponseMsg).TxBytes[1], tx2Bytes)
	}
}

// test stack controller event listener handles RECV_SubmitterProcessDownRequestMsg correctly when no history
func TestRECV_SubmitterProcessDownRequestMsg_NoEntry(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, sharder, endorser, p2pLayer, testDb := initMocksAndDb()

	// setup DLT DB with submitter history using 2 shard/transaction pairs for the submitter/seq
	submitter := []byte("a test submitter")
	seq := uint64(0x21)
	tx1 := TestSignedTransaction("test payload1")
	tx1.Request().SubmitterId = submitter
	tx1.Request().SubmitterSeq = seq
	tx1.Request().ShardId = []byte("shard 1")
	if err := testDb.UpdateSubmitter(tx1); err != nil {
		t.Errorf("Failed to update submitter history for transaction 1: %s", err)
	} else if err := testDb.UpdateShard(tx1); err != nil {
		t.Errorf("Failed to update shard DAG for transaction 1: %s", err)
	} else if err := testDb.AddTx(tx1); err != nil {
		t.Errorf("Failed to save transaction 1: %s", err)
	}
	tx2 := TestSignedTransaction("test payload2")
	tx2.Request().SubmitterId = submitter
	tx2.Request().SubmitterSeq = seq
	tx2.Request().ShardId = []byte("shard 2")
	if err := testDb.UpdateSubmitter(tx2); err != nil {
		t.Errorf("Failed to update submitter history for transaction 2: %s", err)
	} else if err := testDb.UpdateShard(tx2); err != nil {
		t.Errorf("Failed to update shard DAG for transaction 2: %s", err)
	} else if err := testDb.AddTx(tx2); err != nil {
		t.Errorf("Failed to save transaction 2: %s", err)
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
		stack.peerEventsListener(peer, events)
		finished <- struct{}{}
	}()

	// now emit RECV_SubmitterProcessDownRequestMsg event requesting history using submitter's seq that has no history
	req := &SubmitterProcessDownRequestMsg{
		Submitter: submitter,
		Seq:       seq + 1,
	}
	events <- newControllerEvent(RECV_SubmitterProcessDownRequestMsg, req)
	events <- newControllerEvent(SHUTDOWN, nil)

	// wait for event listener to finish
	<-finished

	// check if event listener correctly processed the event to handle submitter's process down request

	// verify that endorser gets called to fetch submitter's known shards/tx
	if !endorser.KnownShardsTxsCalled {
		t.Errorf("Endorser did not get called for submitter/seq history")
	}

	// we should have sent the SubmitterProcessDownResponseMsg back with correct transactions from submitter/seq history
	if !peer.SendCalled {
		t.Errorf("did not send any message to peer")
	} else if peer.SendMsgCode != SubmitterProcessDownResponseMsgCode {
		t.Errorf("Incorrect message code send: %d", peer.SendMsgCode)
	} else if string(peer.SendMsg.(*SubmitterProcessDownResponseMsg).Submitter) != string(req.Submitter) {
		t.Errorf("Incorrect response submitter: %x\nExpected: %x", peer.SendMsg.(*SubmitterProcessDownResponseMsg).Submitter, req.Submitter)
	} else if peer.SendMsg.(*SubmitterProcessDownResponseMsg).Seq != req.Seq {
		t.Errorf("Incorrect response sequence: %x\nExpected: %x", peer.SendMsg.(*SubmitterProcessDownResponseMsg).Seq, req.Seq)
	} else if len(peer.SendMsg.(*SubmitterProcessDownResponseMsg).TxBytes) != 0 {
		t.Errorf("Incorrect number of transactions: %d", len(peer.SendMsg.(*SubmitterProcessDownResponseMsg).TxBytes))
	}
}

// test stack controller event listener handles RECV_SubmitterProcessDownRequestMsg correctly for 0 seq
func TestRECV_SubmitterProcessDownRequestMsg_ZeroSeq(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, sharder, endorser, p2pLayer, testDb := initMocksAndDb()

	// setup DLT DB with submitter history using 1 shard/transaction pairs for the submitter/seq
	submitter := []byte("a test submitter")
	seq := uint64(0x1)
	tx1 := TestSignedTransaction("test payload1")
	tx1.Request().SubmitterId = submitter
	tx1.Request().SubmitterSeq = seq
	tx1.Request().ShardId = []byte("shard 1")
	if err := testDb.UpdateSubmitter(tx1); err != nil {
		t.Errorf("Failed to update submitter history for transaction 1: %s", err)
	} else if err := testDb.UpdateShard(tx1); err != nil {
		t.Errorf("Failed to update shard DAG for transaction 1: %s", err)
	} else if err := testDb.AddTx(tx1); err != nil {
		t.Errorf("Failed to save transaction 1: %s", err)
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
		stack.peerEventsListener(peer, events)
		finished <- struct{}{}
	}()

	// now emit RECV_SubmitterProcessDownRequestMsg event requesting history using submitter's 0 seq
	req := &SubmitterProcessDownRequestMsg{
		Submitter: submitter,
		Seq:       0x00,
	}
	events <- newControllerEvent(RECV_SubmitterProcessDownRequestMsg, req)
	events <- newControllerEvent(SHUTDOWN, nil)

	// wait for event listener to finish
	<-finished

	// check if event listener correctly processed the event to handle submitter's process down request

	// verify that endorser gets called to fetch submitter's known shards/tx
	if !endorser.KnownShardsTxsCalled {
		t.Errorf("Endorser did not get called for submitter/seq history")
	}

	// we should have sent the SubmitterProcessDownResponseMsg back with correct transactions from submitter/seq history
	if !peer.SendCalled {
		t.Errorf("did not send any message to peer")
	} else if peer.SendMsgCode != SubmitterProcessDownResponseMsgCode {
		t.Errorf("Incorrect message code send: %d", peer.SendMsgCode)
	} else if string(peer.SendMsg.(*SubmitterProcessDownResponseMsg).Submitter) != string(req.Submitter) {
		t.Errorf("Incorrect response submitter: %x\nExpected: %x", peer.SendMsg.(*SubmitterProcessDownResponseMsg).Submitter, req.Submitter)
	} else if peer.SendMsg.(*SubmitterProcessDownResponseMsg).Seq != req.Seq {
		t.Errorf("Incorrect response sequence: %x\nExpected: %x", peer.SendMsg.(*SubmitterProcessDownResponseMsg).Seq, req.Seq)
	} else if len(peer.SendMsg.(*SubmitterProcessDownResponseMsg).TxBytes) != 0 {
		t.Errorf("Incorrect number of transactions: %d", len(peer.SendMsg.(*SubmitterProcessDownResponseMsg).TxBytes))
	}
}

// stack controller listner generates RECV_SubmitterProcessDownResponseMsg event for SubmitterProcessDownResponseMsg message
func TestPeerListnerGeneratesEventForSubmitterProcessDownResponseMsg(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, _, _, _ := initMocks()

	// build a mock peer
	mockConn := p2p.TestConn()
	peer := NewMockPeer(mockConn)

	// setup mock connection to send a SubmitterProcessDownResponseMsg followed by clean shutdown
	mockConn.NextMsg(SubmitterProcessDownResponseMsgCode, &SubmitterProcessDownResponseMsg{})
	mockConn.NextMsg(NodeShutdownMsgCode, &NodeShutdown{})

	// setup a test event listener
	events := make(chan controllerEvent, 10)
	finished := checkForEventCode(RECV_SubmitterProcessDownResponseMsg, events)

	// now call stack's listener
	if err := stack.listener(peer, events); err != nil {
		t.Errorf("Transaction processing has errors: %s", err)
	}

	// wait for event listener to process
	result := <-finished

	// check if listener generate correct event
	if !result.seenMsgEvent {
		t.Errorf("Event listener did not generate RECV_SubmitterProcessDownResponseMsg event!!!")
	}
}

// test stack controller event listener handles SubmitterProcessDownResponseMsg correctly when tx's submitter/seq does not match
func TestRECV_SubmitterProcessDownResponseMsg_SubmitterSeqNoMatch(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, sharder, endorser, p2pLayer, testDb := initMocksAndDb()

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
		stack.peerEventsListener(peer, events)
		finished <- struct{}{}
	}()

	// now emit RECV_SubmitterProcessDownResponseMsg event with a transaction that does not match submitter/seq
	msg := NewSubmitterProcessDownResponseMsg(&SubmitterProcessDownRequestMsg{
		Submitter: []byte("some random submitter"),
		Seq:       0xff,
	}, []dto.Transaction{TestSignedTransaction("test payload1")})
	events <- newControllerEvent(RECV_SubmitterProcessDownResponseMsg, msg)
	events <- newControllerEvent(SHUTDOWN, nil)

	// wait for event listener to finish
	<-finished

	// check if event listener correctly processed the event to handle submitter's process down request

	// we should not have processed and broadcast the transaction
	if p2pLayer.DidBroadcast {
		t.Errorf("should not send broadcast mismatched transaction")
	}

	// we should disconnect with peer
	if !peer.DisconnectCalled {
		t.Errorf("should disconnect with peer sending mismatched transaction")
	}
}

// test stack controller event listener handles SubmitterProcessDownResponseMsg correctly
// if local node has a transaction for that shard-id but DOES NOT matches in peer's response, then initiate a double spending alert!!!
func TestRECV_SubmitterProcessDownResponseMsg_DoubleSpend(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, sharder, endorser, p2pLayer, testDb := initMocksAndDb()

	// setup DLT DB with submitter history using 1 shard/transaction pairs for the submitter/seq
	submitter := []byte("a test submitter")
	seq := uint64(0x1)
	tx1 := TestSignedTransaction("test payload1")
	tx1.Request().SubmitterId = submitter
	tx1.Request().SubmitterSeq = seq
	tx1.Request().ShardId = []byte("shard 1")
	if err := testDb.UpdateSubmitter(tx1); err != nil {
		t.Errorf("Failed to update submitter history for transaction 1: %s", err)
	} else if err := testDb.UpdateShard(tx1); err != nil {
		t.Errorf("Failed to update shard DAG for transaction 1: %s", err)
	} else if err := testDb.AddTx(tx1); err != nil {
		t.Errorf("Failed to save transaction 1: %s", err)
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
		stack.peerEventsListener(peer, events)
		finished <- struct{}{}
	}()

	// now emit RECV_SubmitterProcessDownResponseMsg event with a transaction that is double spending
	tx2 := TestSignedTransaction("test payload1")
	*tx2.Request() = *tx1.Request()
	*tx2.Anchor() = *tx1.Anchor()
	tx2.Anchor().Signature = []byte("different Id same tx")
	msg := NewSubmitterProcessDownResponseMsg(&SubmitterProcessDownRequestMsg{
		Submitter: tx1.Request().SubmitterId,
		Seq:       tx1.Request().SubmitterSeq,
	}, []dto.Transaction{tx2})
	events <- newControllerEvent(RECV_SubmitterProcessDownResponseMsg, msg)
	events <- newControllerEvent(SHUTDOWN, nil)

	// wait for event listener to finish
	<-finished

	// check if event listener correctly processed the event to handle submitter's process down request
	// verify that endorser gets called for handling the transaction in response message
	if !endorser.TxHandlerCalled {
		t.Errorf("Endorser did not get called for network transaction")
	}
	if endorser.TxId != tx2.Id() {
		t.Errorf("Endorser transaction does not match network transaction")
	}

	// sharding layer should not have be asked to handle double spending transaction
	if sharder.TxHandlerCalled {
		t.Errorf("DLT stack controller should not call sharding layer for double spending transaction")
	}
	if sharder.LockStateCalled {
		t.Errorf("Controller should not lock world state before transaction processing")
	}
	if sharder.UnlockStateCalled {
		t.Errorf("Controller should not unlock world state after transaction processing")
	}
	if sharder.CommitStateCalled {
		t.Errorf("Controller should not commit world state upon failed transaction processing")
	}

	// we should NOT have broadcasted message
	if p2pLayer.DidBroadcast {
		t.Errorf("Listener should not froward double spending network transaction")
	}

	// we should have emitted a DoubleSpendAlert event
	if len(events) == 0 || (<-events).code != ALERT_DoubleSpend {
		t.Errorf("did not emit ALERT_DoubleSpend")
	}

	// we should NOT have sent any message to peer
	if peer.SendCalled {
		t.Errorf("should not send any message to peer")
	}

	// we should not disconnect here, let double spending event handler take care of decision
	if peer.DisconnectCalled {
		t.Errorf("Listener should not disconnect peer for double spending network transaction")
	}
}

// test stack controller event listener handles SubmitterProcessDownResponseMsg correctly
// if transaction's shard ancestor is unknown, initiate a shard sync with peer for that shard-id
func TestRECV_SubmitterProcessDownResponseMsg_UnknownShardAncestor(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, sharder, endorser, p2pLayer, testDb := initMocksAndDb()

	// setup DLT DB with submitter history using 1 shard/transaction pairs for the submitter/seq
	submitter := []byte("a test submitter")
	seq := uint64(0x1)
	tx1 := TestSignedTransaction("test payload1")
	tx1.Request().SubmitterId = submitter
	tx1.Request().SubmitterSeq = seq
	tx1.Request().ShardId = []byte("shard 1")
	if err := testDb.UpdateSubmitter(tx1); err != nil {
		t.Errorf("Failed to update submitter history for transaction 1: %s", err)
	} else if err := testDb.UpdateShard(tx1); err != nil {
		t.Errorf("Failed to update shard DAG for transaction 1: %s", err)
	} else if err := testDb.AddTx(tx1); err != nil {
		t.Errorf("Failed to save transaction 1: %s", err)
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
		stack.peerEventsListener(peer, events)
		finished <- struct{}{}
	}()

	// now emit RECV_SubmitterProcessDownResponseMsg event with a transaction that has a ShardParent unknown locally
	tx2 := TestSignedTransaction("test payload2")
	tx2.Anchor().ShardParent = dto.RandomHash()
	tx2.Request().SubmitterId = submitter
	tx2.Request().SubmitterSeq = seq
	tx2.Request().ShardId = []byte("shard 2")
	msg := NewSubmitterProcessDownResponseMsg(&SubmitterProcessDownRequestMsg{
		Submitter: tx2.Request().SubmitterId,
		Seq:       tx2.Request().SubmitterSeq,
	}, []dto.Transaction{tx2})
	events <- newControllerEvent(RECV_SubmitterProcessDownResponseMsg, msg)
	events <- newControllerEvent(SHUTDOWN, nil)

	// wait for event listener to finish
	<-finished

	// check if event listener correctly processed the event to handle submitter's process down request
	// when transaction's shard parent is unknown to local node

	// we should have fetched anchor from sharder for requested shard
	if !sharder.SyncAnchorCalled {
		t.Errorf("did not fetch sync anchor from sharder")
	}

	// we should set the peer state to expect ancestors response for requested hash
	if state := peer.GetState(int(RECV_ShardAncestorResponseMsg)); state == nil || state.([64]byte) != tx2.Anchor().ShardParent {
		t.Errorf("controller set expected state to incorrect hash:\n%x\nExpected:\n%x", state, tx2.Anchor().ShardParent)
	}

	// we should have sent the ShardAncestorRequestMsg message to initiate WalkUpStage on remote peer's shard DAG
	if !peer.SendCalled {
		t.Errorf("did not send any message to peer")
	} else if peer.SendMsgCode != ShardAncestorRequestMsgCode {
		t.Errorf("Incorrect message code send: %d", peer.SendMsgCode)
	} else if peer.SendMsg.(*ShardAncestorRequestMsg).StartHash != tx2.Anchor().ShardParent {
		t.Errorf("Incorrect walk up hash: %x\nExpected: %x", peer.SendMsg.(*ShardAncestorRequestMsg).StartHash, tx2.Anchor().ShardParent)
	}

	// verify that endorser does not gets called for handling the transaction in response message
	if endorser.TxHandlerCalled {
		t.Errorf("Endorser should not get called when initiating shard sync")
	}
	// sharding layer should not have be asked to handle transaction triggering shard sync
	if sharder.TxHandlerCalled {
		t.Errorf("DLT stack controller should not call sharding layer to handle transaction triggering shard sync")
	}
	if sharder.LockStateCalled {
		t.Errorf("Controller should not lock world state before transaction processing")
	}
	if sharder.UnlockStateCalled {
		t.Errorf("Controller should not unlock world state after transaction processing")
	}
	if sharder.CommitStateCalled {
		t.Errorf("Controller should not commit world state upon failed transaction processing")
	}
	// we should NOT have broadcasted message
	if p2pLayer.DidBroadcast {
		t.Errorf("Listener should not froward double spending network transaction")
	}
}

// test stack controller event listener handles SubmitterProcessDownResponseMsg correctly
// if local node does not have any transaction for transaction's submitter/seq/shard-id
func TestRECV_SubmitterProcessDownResponseMsg_HappyPath(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, sharder, endorser, p2pLayer, testDb := initMocksAndDb()

	// setup DLT DB with submitter history using 1 shard/transaction pairs for the submitter/seq
	stack.Submit(dto.TestSubmitter().NewRequest("test payload1"))

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
		stack.peerEventsListener(peer, events)
		finished <- struct{}{}
	}()

	// now emit RECV_SubmitterProcessDownResponseMsg event with a transaction is not known locally
	tx2 := TestSignedTransaction("test payload2")
	msg := NewSubmitterProcessDownResponseMsg(&SubmitterProcessDownRequestMsg{
		Submitter: tx2.Request().SubmitterId,
		Seq:       tx2.Request().SubmitterSeq,
	}, []dto.Transaction{tx2})
	events <- newControllerEvent(RECV_SubmitterProcessDownResponseMsg, msg)
	events <- newControllerEvent(SHUTDOWN, nil)

	// wait for event listener to finish
	<-finished

	// check if event listener correctly processed the event to handle submitter's process down request
	// when transaction is unknown to local node

	// we should have broadcasted message
	if !p2pLayer.DidBroadcast {
		t.Errorf("Listener did not froward network transaction")
	}

	// sharding layer should be asked to handle transaction
	if !sharder.TxHandlerCalled {
		t.Errorf("DLT stack controller did not call sharding layer")
	}
	if !sharder.LockStateCalled {
		t.Errorf("Controller did not lock world state before transaction processing")
	}
	if !sharder.UnlockStateCalled {
		t.Errorf("Controller did not unlock world state after transaction processing")
	}
	if !sharder.CommitStateCalled {
		t.Errorf("Controller should commit world state upon successfull transaction processing")
	}

	// verify that endorser gets called for network message
	if !endorser.TxHandlerCalled {
		t.Errorf("Endorser did not get called for network transaction")
	}
	if endorser.TxId != tx2.Id() {
		t.Errorf("Endorser transaction does not match network transaction")
	}

	// we should have sent the SubmitterProcessDownRequestMsg message to fetch submitter's history for next sequence from peer
	if !peer.SendCalled {
		t.Errorf("did not send any message to peer")
	} else if peer.SendMsgCode != SubmitterProcessDownRequestMsgCode {
		t.Errorf("Incorrect message code send: %d", peer.SendMsgCode)
	} else if string(peer.SendMsg.(*SubmitterProcessDownRequestMsg).Submitter) != string(msg.Submitter) {
		t.Errorf("Incorrect submitter: %x\nExpected: %x", peer.SendMsg.(*SubmitterProcessDownRequestMsg).Submitter, msg.Submitter)
	} else if peer.SendMsg.(*SubmitterProcessDownRequestMsg).Seq != msg.Seq+1 {
		t.Errorf("Incorrect sequence: %d\nExpected: %d", peer.SendMsg.(*SubmitterProcessDownRequestMsg).Seq, msg.Seq+1)
	}
}

// test stack controller event listener handles SubmitterProcessDownResponseMsg correctly
// if response does not have any transactions then EndOfSync
func TestRECV_SubmitterProcessDownResponseMsg_EndOfSync(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, sharder, endorser, p2pLayer, testDb := initMocksAndDb()

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
		stack.peerEventsListener(peer, events)
		finished <- struct{}{}
	}()

	// now emit RECV_SubmitterProcessDownResponseMsg event with no transactions
	msg := NewSubmitterProcessDownResponseMsg(&SubmitterProcessDownRequestMsg{
		Submitter: []byte("some submitter"),
		Seq:       0x23,
	}, []dto.Transaction{})
	events <- newControllerEvent(RECV_SubmitterProcessDownResponseMsg, msg)
	events <- newControllerEvent(SHUTDOWN, nil)

	// wait for event listener to finish
	<-finished

	// check if event listener correctly processed the event to handle submitter's process down request
	// when there are no transactions in response
	// sharding layer should not have be asked to handle double spending transaction
	if sharder.TxHandlerCalled {
		t.Errorf("DLT stack controller should not call sharding layer for end of sync")
	}
	if sharder.LockStateCalled {
		t.Errorf("Controller should not lock world state for end of sync")
	}
	if sharder.UnlockStateCalled {
		t.Errorf("Controller should not unlock world state for end of sync")
	}
	if sharder.CommitStateCalled {
		t.Errorf("Controller should not commit world state for end of sync")
	}

	// we should NOT have broadcasted message
	if p2pLayer.DidBroadcast {
		t.Errorf("Listener should not froward for end of sync")
	}
}
