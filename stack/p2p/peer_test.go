package p2p

import (
    "testing"
    "fmt"
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
	peer.Send(0, struct{}{})
	if conn.WriteCount != 1 {
		t.Errorf("Failed to send message to Peer via connection")
	}
}

func TestDEVp2pPeerReadMsg(t *testing.T) {
	conn := TestConn()
	peer := NewDEVp2pPeer(TestMockPeer("test peer"), conn)
	m, _ := peer.ReadMsg()
	if conn.ReadCount != 1 {
		t.Errorf("Failed to read message from Peer via connection")
	}
	if m.(*msg) == nil || m.(*msg).p2pMsg == nil {
		t.Errorf("message wrapper not initialized correctly")
	}
}
