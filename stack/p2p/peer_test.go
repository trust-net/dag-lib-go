package p2p

import (
	"fmt"
	"github.com/trust-net/dag-lib-go/stack/dto"
	"testing"
)

func TestDEVp2pPeerInstance(t *testing.T) {
	var peer Peer
	peer = NewDEVp2pPeer(TestDEVp2pPeer("test peer"), TestConn())
	if impl, ok := peer.(*peerDEVp2p); ok {
		fmt.Printf("Peer type: %T\n", impl)
	} else {
		t.Errorf("Failed to cast Peer into DEVp2p Peer implementation")
	}
	if peer.Status() != Connected {
		t.Errorf("Failed to connect with Peer")
	}
}

func TestDEVp2pPeerID(t *testing.T) {
	p2p := TestMockPeer("test peer")
	peer := NewDEVp2pPeer(p2p, TestConn())
	peer.ID()
	if p2p.IdCount != 1 {
		t.Errorf("Failed to get ID from Peer")
	}
}

func TestDEVp2pPeerName(t *testing.T) {
	p2p := TestMockPeer("test peer")
	peer := NewDEVp2pPeer(p2p, TestConn())
	peer.Name()
	if p2p.NameCount != 1 {
		t.Errorf("Failed to get Name from Peer")
	}
}

func TestDEVp2pPeerRemoteAddr(t *testing.T) {
	p2p := TestMockPeer("test peer")
	peer := NewDEVp2pPeer(p2p, TestConn())
	peer.RemoteAddr()
	if p2p.RemoteCount != 1 {
		t.Errorf("Failed to get remote address from Peer")
	}
}

func TestDEVp2pPeerLocalAddr(t *testing.T) {
	p2p := TestMockPeer("test peer")
	peer := NewDEVp2pPeer(p2p, TestConn())
	peer.LocalAddr()
	if p2p.LocalCount != 1 {
		t.Errorf("Failed to get local address from Peer")
	}
}

func TestDEVp2pPeerDisconnect(t *testing.T) {
	p2p := TestMockPeer("test peer")
	peer := NewDEVp2pPeer(p2p, TestConn())
	peer.Disconnect()
	if peer.Status() != Disconnected {
		t.Errorf("Peer status not updated")
	}
	if p2p.DisconnectCount != 1 {
		t.Errorf("Failed to disconnect with Peer")
	}
}

func TestDEVp2pPeerString(t *testing.T) {
	p2p := TestMockPeer("test peer")
	peer := NewDEVp2pPeer(p2p, TestConn())
	peer.String()
	if p2p.StringCount != 1 {
		t.Errorf("Failed to get string representation from Peer")
	}
}

func TestDEVp2pPeerSend(t *testing.T) {
	conn := TestConn()
	peer := NewDEVp2pPeer(TestMockPeer("test peer"), conn)
	peer.Send([]byte("id1"), uint64(0), struct{}{})
	if conn.WriteCount != 1 {
		t.Errorf("Failed to send message to Peer via connection")
	}
}

func TestDEVp2pPeerSendSeen(t *testing.T) {
	conn := TestConn()
	peer := NewDEVp2pPeer(TestMockPeer("test peer"), conn)
	peer.Seen([]byte("id1"))
	peer.Send([]byte("id1"), uint64(0), struct{}{})
	if conn.WriteCount != 0 {
		t.Errorf("Did not skip seen message from sending again")
	}
}

func TestDEVp2pPeerReadMsg(t *testing.T) {
	conn := TestConn()
	conn.NextMsg(0, &struct{}{})
	peer := NewDEVp2pPeer(TestMockPeer("test peer"), conn)
	m, _ := peer.ReadMsg()
	if conn.ReadCount != 1 {
		t.Errorf("Failed to read message from Peer via connection")
	}
	if m.(*msg) == nil || m.(*msg).p2pMsg == nil {
		t.Errorf("message wrapper not initialized correctly")
	}
}

func TestSetState(t *testing.T) {
	conn := TestConn()
	peer := NewDEVp2pPeer(TestMockPeer("test peer"), conn)
	if err := peer.SetState(0, "test set state"); err != nil {
		t.Errorf("Failed to set state: %s", err)
	}
}

func TestGetState(t *testing.T) {
	conn := TestConn()
	peer := NewDEVp2pPeer(TestMockPeer("test peer"), conn)
	// test getting a non existing stateId
	if state := peer.GetState(999); state != nil {
		t.Errorf("Did not expect state for non existent id")
	}

	// test getting a composite state
	type testType struct {
		sData string
		iData int
		slice []byte
	}
	peer.SetState(33, testType{
		sData: "test string",
		iData: 0x34,
		slice: []byte{0x03, 0x04, 0x3e},
	})
	value := peer.GetState(33)
	if value == nil {
		t.Errorf("Failed to get state")
	} else if state, ok := value.(testType); !ok {
		t.Errorf("incorrect type of saved state")
	} else {
		if state.sData != "test string" {
			t.Errorf("incorrect string state value: %s", state.sData)
		}
		if state.iData != 0x34 {
			t.Errorf("incorrect integer state value: %d", state.iData)
		}
		if string(state.slice) != string([]byte{0x03, 0x04, 0x3e}) {
			t.Errorf("incorrect slice state value: %x", state.slice)
		}
	}
}

func TestToBeFetchedStackPush(t *testing.T) {
	conn := TestConn()
	peer := NewDEVp2pPeer(TestMockPeer("test peer"), conn)
	if err := peer.ToBeFetchedStackPush(dto.TestSignedTransaction("test data")); err != nil {
		t.Errorf("Failed to push transaction: %s", err)
	}
}

func TestToBeFetchedStackPop_HasTx(t *testing.T) {
	conn := TestConn()
	peer := NewDEVp2pPeer(TestMockPeer("test peer"), conn)
	// push few transactions to bottom
	peer.ToBeFetchedStackPush(dto.TestSignedTransaction("test data"))
	peer.ToBeFetchedStackPush(dto.TestSignedTransaction("test data"))
	// now push one more on top
	tx := dto.TestSignedTransaction("test data")
	peer.ToBeFetchedStackPush(tx)

	// pop a transaction, it should be the one pushed last
	if popped := peer.ToBeFetchedStackPop(); popped == nil {
		t.Errorf("Failed to pop transaction")
	} else if popped.Id() != tx.Id() {
		t.Errorf("Failed to pop the top transaction")
	}
}

func TestToBeFetchedStackPop_HasNoTx(t *testing.T) {
	conn := TestConn()
	peer := NewDEVp2pPeer(TestMockPeer("test peer"), conn)

	// pop a transaction from empty stack, it should be nil
	if popped := peer.ToBeFetchedStackPop(); popped != nil {
		t.Errorf("Unexpected popped transaction")
	}
}
