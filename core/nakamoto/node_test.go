package nakamoto

import (
	"testing"
	"github.com/liamzebedee/tinychain-go/core"
)

func TestNewNode(t *testing.T) {
	// DAG.
	dag, _, _ := newBlockdag()

	// Miner.
	minerWallet, err := core.CreateRandomWallet()
	if err != nil {
		t.Fatalf("Failed to create miner wallet: %s", err)
	}

	miner := NewMiner(dag, minerWallet)

	// Peer.
	peer := NewPeerCore(PeerConfig{port: "8080"})

	// Create the node.
	node := NewNode(dag, miner, peer)

	// Start the node.
	node.Start()
}