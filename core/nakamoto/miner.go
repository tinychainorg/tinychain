package nakamoto

import (
	"math/big"
	"fmt"
	"time"
	"github.com/liamzebedee/tinychain-go/core"
	"golang.org/x/text/message"
	"golang.org/x/text/language"
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

type POWPuzzle struct {
	block *RawBlock
	startNonce big.Int
	target big.Int
	solution big.Int
}

func MineWithStatus(hashrateChannel chan float64, solutionChannel chan POWPuzzle, puzzleChannel chan POWPuzzle) (big.Int, error) {
	// Execute in 3s increments.
	lastHashrateMeasurement := Timestamp()
	numHashes := 0

	// Measure hashrate.
	go (func() {
		for {
			// Wait 3s.
			<-time.After(3 * time.Second)

			// Print iterations using commas.
			p := message.NewPrinter(language.English)
			p.Printf("Hashes: %d\n", numHashes)

			// Check if 3s has elapsed since last time.
			now := Timestamp()
			duration := now - lastHashrateMeasurement
			hashrate := float64(numHashes) / float64(duration/1000)
			hashrateChannel <- hashrate
			numHashes = 0
			lastHashrateMeasurement = now
		}
	})()

	for {
		var i uint64 = 0
		fmt.Println("waiting for new puzzle")
		puzzle := <- puzzleChannel
		block := puzzle.block
		nonce := puzzle.startNonce
		target := puzzle.target
		fmt.Printf("New puzzle block=%s target=%s\n", block.HashStr(), target.String())

		for {
			numHashes++
			i++
	
			// Increment nonce.
			nonce.Add(&nonce, big.NewInt(1))
			block.SetNonce(nonce)
	
			// Hash.
			h := block.Hash()
			hash := new(big.Int).SetBytes(h[:])
	
			// Check solution: hash < target.
			if hash.Cmp(&target) == -1 {
				// fmt.Printf("Solved in %d iterations\n", i)
				// fmt.Printf("Hash: %x\n", hash.String())
				// fmt.Printf("Nonce: %s\n", nonce.String())
				fmt.Println("Puzzle solved")

				puzzle.solution = nonce
				solutionChannel <- puzzle
				break
			}

			// Check if new puzzle has been received.
			select {
			case newPuzzle := <- puzzleChannel:
				fmt.Println("Received new puzzle")
				puzzle = newPuzzle
				block = puzzle.block
				nonce = puzzle.startNonce
				target = puzzle.target
				fmt.Printf("New puzzle block=%s target=%s\n", block.HashStr(), target.String())
			default:
				// Do nothing.
			}
		}
	}
}

func (node *Node) MakeNewPuzzle() (POWPuzzle) {
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

	puzzle := POWPuzzle{
		block: &raw,
		startNonce: *big.NewInt(0),
		target: difficulty,
	}
	return puzzle
}

func (node *Node) Start() {
	// The next tip channel.
	// next_tip := make(chan Block)
	// block_solutions := make(chan Block)
	hashrateChannel := make(chan float64, 1)
	puzzleChannel := make(chan POWPuzzle, 1)
	solutionChannel := make(chan POWPuzzle, 1)

	go MineWithStatus(hashrateChannel, solutionChannel, puzzleChannel)

	puzzleChannel <- node.MakeNewPuzzle()
	for {
		select {
		case hashrate := <-hashrateChannel:
			// Print iterations using commas.
			p := message.NewPrinter(language.English)
			p.Printf("Hashrate: %.2f H/s\n", hashrate)
		case puzzle := <-solutionChannel:
			fmt.Println("Received solution")

			raw := puzzle.block
			solution := puzzle.solution
			raw.SetNonce(solution)

			fmt.Printf("Solution: hash=%s nonce=%s\n", Bytes32ToString(raw.Hash()), solution.String())

			// Ingest block.
			err := node.dag.IngestBlock(*raw)
			if err != nil {
				fmt.Errorf("Failed to ingest block: %s\n", err)
			}

			// Gossip block.
			fmt.Println("Making new puzzle")
			fmt.Println("New puzzle ready")
			puzzleChannel <- node.MakeNewPuzzle()

			// fmt.Printf("Solution: height=%d hash=%s nonce=%s\n", curr_height, Bytes32ToString(raw.Hash()), solution.String())
		}
	}
	
}
