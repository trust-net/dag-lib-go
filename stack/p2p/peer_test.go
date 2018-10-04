package p2p

import (
    "testing"
    "fmt"
)

func TestDEVp2pPeerInstance(t *testing.T) {
	var peer Peer
	peer = NewDEVp2pPeer(testDEVp2pPeer("test peer"), testConn())
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
	p2p := testMockPeer("test peer")
	peer := NewDEVp2pPeer(p2p, testConn())
	peer.ID()
	if p2p.idCount != 1 {
		t.Errorf("Failed to get ID from Peer")
	}
}

func TestDEVp2pPeerName(t *testing.T) {
	p2p := testMockPeer("test peer")
	peer := NewDEVp2pPeer(p2p, testConn())
	peer.Name()
	if p2p.nameCount != 1 {
		t.Errorf("Failed to get Name from Peer")
	}
}

func TestDEVp2pPeerRemoteAddr(t *testing.T) {
	p2p := testMockPeer("test peer")
	peer := NewDEVp2pPeer(p2p, testConn())
	peer.RemoteAddr()
	if p2p.remoteCount != 1 {
		t.Errorf("Failed to get remote address from Peer")
	}
}

func TestDEVp2pPeerLocalAddr(t *testing.T) {
	p2p := testMockPeer("test peer")
	peer := NewDEVp2pPeer(p2p, testConn())
	peer.LocalAddr()
	if p2p.localCount != 1 {
		t.Errorf("Failed to get local address from Peer")
	}
}

func TestDEVp2pPeerDisconnect(t *testing.T) {
	p2p := testMockPeer("test peer")
	peer := NewDEVp2pPeer(p2p, testConn())
	peer.Disconnect()
	if peer.Status() != Disconnected {
		t.Errorf("Peer status not updated")
	}
	if p2p.disconnectCount != 1 {
		t.Errorf("Failed to disconnect with Peer")
	}
}

func TestDEVp2pPeerString(t *testing.T) {
	p2p := testMockPeer("test peer")
	peer := NewDEVp2pPeer(p2p, testConn())
	peer.String()
	if p2p.stringCount != 1 {
		t.Errorf("Failed to get string representation from Peer")
	}
}

func TestDEVp2pPeerSend(t *testing.T) {
	conn := testConn()
	peer := NewDEVp2pPeer(testMockPeer("test peer"), conn)
	peer.Send(0, struct{}{})
	if conn.writeCount != 1 {
		t.Errorf("Failed to send message to Peer via connection")
	}
}
