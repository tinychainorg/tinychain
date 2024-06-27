package nakamoto

import (
	"testing"
)

func TestStartPeer(t *testing.T) {
	peer := NewPeerServer(PeerConfig{port: "8080"})
	peer.Start()
}