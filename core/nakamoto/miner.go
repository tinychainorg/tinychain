package nakamoto

import (
	"math/big"
	"fmt"
	"github.com/liamzebedee/tinychain-go/core"
)

type Node struct {
	dag BlockDAG
	minerWallet *core.Wallet
}

func NewNode(dag BlockDAG, minerWallet *core.Wallet) *Node {
	return &Node{
		dag: dag,
		minerWallet: minerWallet,
	}
}

func MakeCoinbaseTx(wallet *core.Wallet) RawTransaction {
	// Construct coinbase tx.
	tx := RawTransaction{
		Sig: [64]byte{},
		FromPubkey: [65]byte{},
		Data: []byte("coinbase"),
	}
	tx.FromPubkey = wallet.PubkeyBytes()
	envelope := tx.Envelope()
	sig, err := wallet.Sign(envelope)
	if err != nil {
		panic(err)
	}
	copy(tx.Sig[:], sig)
	return tx
}

func (node *Node) Start() {
	// The next tip channel.
	// next_tip := make(chan Block)
	// block_solutions := make(chan Block)

	for {
		current_tip, err := node.dag.GetCurrentTip()
		if err != nil {
			// fmt.Fatalf("Failed to get current tip: %s", err)
			panic(err)
		}

		// Construct coinbase tx.
		tx := MakeCoinbaseTx(node.minerWallet)

		// Construct block template for mining.
		raw := RawBlock{
			ParentHash: current_tip.Hash,
			Timestamp: Timestamp(),
			NumTransactions: 1,
			TransactionsMerkleRoot: [32]byte{},
			Nonce: [32]byte{},
			Transactions: []RawTransaction{
				tx,
			},
		}
		raw.TransactionsMerkleRoot = core.ComputeMerkleHash([][]byte{tx.Envelope()})

		// Mine the POW solution.
		curr_height := current_tip.Height + 1

		// First get the right epoch.
		var difficulty big.Int
		epoch, err := node.dag.GetEpochForBlockHash(raw.ParentHash)
		if err != nil {
			// t.Fatalf("Failed to get epoch for block hash: %s", err)
			panic(err)
		}
		if curr_height % node.dag.consensus.EpochLengthBlocks == 0 {
			difficulty = RecomputeDifficulty(epoch.StartTime, raw.Timestamp, epoch.Difficulty, node.dag.consensus.TargetEpochLengthMillis, node.dag.consensus.EpochLengthBlocks, curr_height)
		} else {
			difficulty = epoch.Difficulty
		}

		solution, err := SolvePOW(raw, *big.NewInt(0), difficulty, 0)
		if err != nil {
			// t.Fatalf("Failed to solve POW: %s", err)
			panic(err)
		}
		fmt.Printf("Solution: height=%d hash=%s nonce=%s\n", curr_height, Bytes32ToString(raw.Hash()), solution.String())
		raw.SetNonce(solution)

		// Ingest block.
		err = node.dag.IngestBlock(raw)
		if err != nil {
			// t.Fatalf("Failed to ingest block: %s", err)
			panic(err)
		}
	}
	
	
	// setMining := make(chan bool)
	// newSolution := make(chan string)

	// isMining := false

	// for {
    //     select {
    //     case isMining = <-setMining:
    //         fmt.Println("startStopSignal", status)
	// 		break;
		
    //     case msg2 := <-signals:
    //         fmt.Println("received", msg2)
		
    //     }
    // }
}
