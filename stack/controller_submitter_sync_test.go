// Copyright 2018-2019 The trust-net Authors
// Controller interface and implementation for DLT Statck
package stack

import (
	"github.com/trust-net/dag-lib-go/log"
	"github.com/trust-net/dag-lib-go/stack/dto"
	"github.com/trust-net/dag-lib-go/stack/p2p"
	"testing"
)

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

	// submit a transactions to add ancestor to local shard's Anchor
	tx := TestSignedTransaction("test payload1")
	stack.Submit(tx)
	nextAnchor := stack.Anchor(tx.Anchor().Submitter, 0x02, tx.Id())
	if nextAnchor == nil {
		t.Errorf("Failed to get anchor for next sequence")
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

	// now emit RECV_SubmitterWalkUpRequestMsg event requesting history using submitter's nextAnchor
	req := NewSubmitterWalkUpRequestMsg(nextAnchor)
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
	} else if string(peer.SendMsg.(*SubmitterWalkUpResponseMsg).Shards[0]) != string(tx.Anchor().ShardId) {
		t.Errorf("Incorrect shard in response: %x\nExpected: %x", peer.SendMsg.(*SubmitterWalkUpResponseMsg).Shards[0], tx.Anchor().ShardId)
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
	tx := TestSignedTransaction("test payload1")
	stack.Submit(tx)
	// build a SubmitterWalkUpResponseMsg that has same submitter, seq but different shard
	nextAnchor := stack.Anchor(tx.Anchor().Submitter, 0x02, tx.Id())
	msg := NewSubmitterWalkUpResponseMsg(NewSubmitterWalkUpRequestMsg(nextAnchor))
	msg.Transactions = [][64]byte{dto.RandomHash()}
	msg.Shards = [][]byte{[]byte("a different shard")}
	p2pLayer.Reset()
	sharder.Reset()
	endorser.Reset()
	testDb.Reset()

	// build a mock peer
	mockConn := p2p.TestConn()
	peer := NewMockPeer(mockConn)
	// set the peer state to expect the message ID
	peer.SetState(int(RECV_SubmitterWalkUpResponseMsg), msg.Id())

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

	// we should have set the peer state
	if data := peer.GetState(int(RECV_SubmitterWalkUpResponseMsg)); data == nil {
		t.Errorf("controller did not save last hash for walk up response message")
	} else if state, ok := data.([]byte); !ok {
		t.Errorf("controller saved incorrect state type: %T", data)
	} else if string(state) != string(peer.SendMsgId) {
		t.Errorf("controller saved incorrect hash:\n%x\nExpected:\n%x", state, peer.SendMsgId)
	}

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
	tx := TestSignedTransaction("test payload1")
	stack.Submit(tx)
	// build a SubmitterWalkUpResponseMsg that has same submitter, seq, shard but different transaction
	nextAnchor := stack.Anchor(tx.Anchor().Submitter, 0x02, tx.Id())
	msg := NewSubmitterWalkUpResponseMsg(NewSubmitterWalkUpRequestMsg(nextAnchor))
	msg.Transactions = [][64]byte{dto.RandomHash()}
	msg.Shards = [][]byte{nextAnchor.ShardId}
	p2pLayer.Reset()
	sharder.Reset()
	endorser.Reset()
	testDb.Reset()

	// build a mock peer
	mockConn := p2p.TestConn()
	peer := NewMockPeer(mockConn)
	// set the peer state to expect the message ID
	peer.SetState(int(RECV_SubmitterWalkUpResponseMsg), msg.Id())

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

	// we should have initiate a double spend resolution

	// we should have set the peer state to nil value
	if data := peer.GetState(int(RECV_SubmitterWalkUpResponseMsg)); data != nil {
		t.Errorf("controller did not reset state for walk up response message")
	}

	// we should have emitted a DoubleSpendAlert event
	if len(events) == 0 || (<-events).code != ALERT_DoubleSpend {
		t.Errorf("did not emit ALERT_DoubleSpend")
	}

	// we should NOT have sent any message to peer
	if peer.SendCalled {
		t.Errorf("should not send any message to peer")
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
	tx1.Anchor().Submitter = submitter
	tx1.Anchor().SubmitterSeq = seq
	tx1.Anchor().ShardId = []byte("shard 1")
	if err := testDb.UpdateSubmitter(tx1); err != nil {
		t.Errorf("Failed to update submitter history for transaction 1: %s", err)
	}
	// build another shard/transaction pair that is NOT in local DLT DB
	tx2 := TestSignedTransaction("test payload2")
	tx2.Anchor().Submitter = submitter
	tx2.Anchor().SubmitterSeq = seq
	tx2.Anchor().ShardId = []byte("shard 2")
	// build a SubmitterWalkUpResponseMsg that has above submitter/seq and both shard/transaction pairs
	msg := &SubmitterWalkUpResponseMsg{
		Submitter: submitter,
		Seq:       seq,
	}
	msg.Transactions = [][64]byte{tx1.Id(), tx2.Id()}
	msg.Shards = [][]byte{tx1.Anchor().ShardId, tx2.Anchor().ShardId}
	p2pLayer.Reset()
	sharder.Reset()
	endorser.Reset()
	testDb.Reset()

	// build a mock peer
	mockConn := p2p.TestConn()
	peer := NewMockPeer(mockConn)
	// set the peer state to expect the message ID
	peer.SetState(int(RECV_SubmitterWalkUpResponseMsg), msg.Id())

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

	// we should have set the peer state
	if data := peer.GetState(int(RECV_SubmitterWalkUpResponseMsg)); data == nil {
		t.Errorf("controller did not save last hash for walk up response message")
	} else if state, ok := data.([]byte); !ok {
		t.Errorf("controller saved incorrect state type: %T", data)
	} else if string(state) != string(peer.SendMsgId) {
		t.Errorf("controller saved incorrect hash:\n%x\nExpected:\n%x", state, peer.SendMsgId)
	}

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
	tx := TestSignedTransaction("test payload1")
	stack.Submit(tx)
	// build a SubmitterWalkUpResponseMsg that has same submitter, seq, shard and transaction
	nextAnchor := stack.Anchor(tx.Anchor().Submitter, 0x02, tx.Id())
	msg := NewSubmitterWalkUpResponseMsg(NewSubmitterWalkUpRequestMsg(nextAnchor))
	msg.Transactions = [][64]byte{tx.Id()}
	msg.Shards = [][]byte{nextAnchor.ShardId}
	p2pLayer.Reset()
	sharder.Reset()
	endorser.Reset()
	testDb.Reset()

	// build a mock peer
	mockConn := p2p.TestConn()
	peer := NewMockPeer(mockConn)
	// set the peer state to expect the message ID
	peer.SetState(int(RECV_SubmitterWalkUpResponseMsg), msg.Id())

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
	if data := peer.GetState(int(RECV_SubmitterWalkUpResponseMsg)); data != nil {
		t.Errorf("controller did not reset state for walk up response message")
	}

	// we should have set state to validate SubmitterProcessDownResponseMsg when received
	if data := peer.GetState(int(RECV_SubmitterProcessDownResponseMsg)); data == nil {
		t.Errorf("controller did not save hash for validating SubmitterProcessDownResponseMsg")
	} else if state, ok := data.([]byte); !ok {
		t.Errorf("controller saved incorrect state type: %T", data)
	} else if string(state) != string(NewSubmitterProcessDownRequestMsg(msg).Id()) {
		t.Errorf("controller saved incorrect hash:\n%x\nExpected:\n%x", state, NewSubmitterProcessDownRequestMsg(msg).Id())
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

// test stack controller event listener handles RECV_SubmitterWalkUpResponseMsg correctly when no pairs in response for non 0 seq
func TestRECV_SubmitterWalkUpResponseMsg_ZeroPairsNonZeroSeq(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, sharder, endorser, p2pLayer, testDb := initMocksAndDb()

	// setup DLT DB with submitter history using 1 shard/transaction pair for the submitter/seq
	submitter := []byte("a test submitter")
	seq := uint64(0x21)
	tx1 := TestSignedTransaction("test payload1")
	tx1.Anchor().Submitter = submitter
	tx1.Anchor().SubmitterSeq = seq
	tx1.Anchor().ShardId = []byte("shard 1")
	if err := testDb.UpdateSubmitter(tx1); err != nil {
		t.Errorf("Failed to update submitter history for transaction 1: %s", err)
	}
	// build a SubmitterWalkUpResponseMsg that has above submitter/seq bot no shard/transaction pairs
	msg := &SubmitterWalkUpResponseMsg{
		Submitter: submitter,
		Seq:       seq,
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
	// set the peer state to expect the message ID
	peer.SetState(int(RECV_SubmitterWalkUpResponseMsg), msg.Id())

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
	tx1.Anchor().Submitter = submitter
	tx1.Anchor().SubmitterSeq = seq
	tx1.Anchor().ShardId = []byte("shard 1")
	if err := testDb.UpdateSubmitter(tx1); err != nil {
		t.Errorf("Failed to update submitter history for transaction 1: %s", err)
	}
	// build a SubmitterWalkUpResponseMsg that has above submitter/seq but mismatch in count of shard/transaction pairs
	msg := &SubmitterWalkUpResponseMsg{
		Submitter: submitter,
		Seq:       seq,
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
	// set the peer state to expect the message ID
	peer.SetState(int(RECV_SubmitterWalkUpResponseMsg), msg.Id())

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
func TestRECV_SubmitterWalkUpResponseMsg_StateValidationFailed(t *testing.T) {
	// create a DLT stack instance with registered app and initialized mocks
	stack, sharder, endorser, p2pLayer, testDb := initMocksAndDb()

	// setup DLT DB with submitter history using 1 shard/transaction pair for the submitter/seq
	submitter := []byte("a test submitter")
	seq := uint64(0x21)
	tx1 := TestSignedTransaction("test payload1")
	tx1.Anchor().Submitter = submitter
	tx1.Anchor().SubmitterSeq = seq
	tx1.Anchor().ShardId = []byte("shard 1")
	if err := testDb.UpdateSubmitter(tx1); err != nil {
		t.Errorf("Failed to update submitter history for transaction 1: %s", err)
	}
	// build a SubmitterWalkUpResponseMsg that has above submitter/seq/shard/tx
	msg := &SubmitterWalkUpResponseMsg{
		Submitter: submitter,
		Seq:       seq,
	}
	msg.Transactions = [][64]byte{tx1.Id()}
	msg.Shards = [][]byte{tx1.Anchor().ShardId}
	p2pLayer.Reset()
	sharder.Reset()
	endorser.Reset()
	testDb.Reset()

	// build a mock peer
	mockConn := p2p.TestConn()
	peer := NewMockPeer(mockConn)
	// set the peer state to a value different from message ID
	peer.SetState(int(RECV_SubmitterWalkUpResponseMsg), []byte("some random value"))

	log.SetLogLevel(log.DEBUG)
	defer log.SetLogLevel(log.NONE)

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
