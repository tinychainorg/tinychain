package nakamoto

import (
	"testing"
	"github.com/liamzebedee/tinychain-go/core"
)

func TestMiner(t *testing.T) {
	dag, _, _ := newBlockdag()
	minerWallet, err := core.CreateRandomWallet()
	if err != nil {
		t.Fatalf("Failed to create miner wallet: %s", err)
	}

	miner := NewNode(dag, minerWallet)
	miner.Start()
}