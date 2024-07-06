package nakamoto

import (
	"math/big"
	"time"

	"sync"

	"github.com/liamzebedee/tinychain-go/core"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var minerLog = NewLogger("miner", "")

type Miner struct {
	dag         BlockDAG
	minerWallet *core.Wallet
	IsRunning   bool

	// Mutex.
	mutex sync.Mutex

	OnBlockSolution func(block RawBlock)
}

func NewMiner(dag BlockDAG, minerWallet *core.Wallet) *Miner {
	return &Miner{
		dag:         dag,
		minerWallet: minerWallet,
		IsRunning:   false,
		mutex:       sync.Mutex{},
	}
}

func MakeCoinbaseTx(wallet *core.Wallet) RawTransaction {
	// Construct coinbase tx.
	tx := RawTransaction{
		Version:    1,
		Sig:        [64]byte{},
		FromPubkey: wallet.PubkeyBytes(),
		ToPubkey:   wallet.PubkeyBytes(),
		Amount:     1000000000,
		Fee:        0,
		Nonce:      0,
	}
	envelope := tx.Envelope()
	sig, err := wallet.Sign(envelope)
	if err != nil {
		panic(err)
	}
	copy(tx.Sig[:], sig)
	return tx
}

type POWPuzzle struct {
	block      *RawBlock
	startNonce big.Int
	target     big.Int
	solution   big.Int
}

func MineWithStatus(hashrateChannel chan float64, solutionChannel chan POWPuzzle, puzzleChannel chan POWPuzzle) (big.Int, error) {
	// Execute in 3s increments.
	lastHashrateMeasurement := Timestamp()
	numHashes := 0

	// Routine: Measure hashrate.
	go (func() {
		for {
			// Wait 3s.
			<-time.After(3 * time.Second)

			// Print iterations using commas.
			// p := message.NewPrinter(language.English)
			// p.Printf("Hashes: %d\n", numHashes)

			// Check if 3s has elapsed since last time.
			now := Timestamp()
			duration := now - lastHashrateMeasurement
			hashrate := float64(numHashes) / float64(duration/1000)
			hashrateChannel <- hashrate
			numHashes = 0
			lastHashrateMeasurement = now
		}
	})()

	// Routine: Mine.
	for {
		var i uint64 = 0
		minerLog.Println("Waiting for new puzzle")
		puzzle := <-puzzleChannel
		block := puzzle.block
		nonce := puzzle.startNonce
		target := puzzle.target
		minerLog.Printf("New puzzle block=%s target=%s\n", block.HashStr(), target.String())

		// Loop: mine 1 hash.
		for {
			numHashes++
			i++

			// Increment nonce.
			nonce.Add(&nonce, big.NewInt(1))
			block.SetNonce(nonce)

			// Hash.
			h := block.Hash()
			guess := new(big.Int).SetBytes(h[:])

			// minerLog.Printf("hash block=%s i=%d\n", Bytes32ToString(h), i)

			// Check solution: hash < target.
			if guess.Cmp(&target) == -1 {
				minerLog.Printf("Puzzle solved: iterations=%d\n", i)

				puzzle.solution = nonce
				solutionChannel <- puzzle
				break
			}

			// Check if new puzzle has been received.
			select {
			case newPuzzle := <-puzzleChannel:
				puzzle = newPuzzle
				block = puzzle.block
				nonce = puzzle.startNonce
				target = puzzle.target
				minerLog.Printf("New puzzle block=%s target=%s\n", block.HashStr(), target.String())
			default:
				// Do nothing.
			}
		}
	}
}

func (node *Miner) MakeNewPuzzle() POWPuzzle {
	current_tip, err := node.dag.GetLatestTip()
	if err != nil {
		// fmt.Fatalf("Failed to get current tip: %s", err)
		panic(err)
	}

	// Construct coinbase tx.
	tx := MakeCoinbaseTx(node.minerWallet)

	// Construct block template for mining.
	raw := RawBlock{
		ParentHash:             current_tip.Hash,
		ParentTotalWork:        BigIntToBytes32(current_tip.AccumulatedWork),
		Timestamp:              Timestamp(),
		NumTransactions:        1,
		TransactionsMerkleRoot: [32]byte{},
		Nonce:                  [32]byte{},
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
	if curr_height%node.dag.consensus.EpochLengthBlocks == 0 {
		difficulty = RecomputeDifficulty(epoch.StartTime, raw.Timestamp, epoch.Difficulty, node.dag.consensus.TargetEpochLengthMillis, node.dag.consensus.EpochLengthBlocks, curr_height)
	} else {
		difficulty = epoch.Difficulty
	}

	puzzle := POWPuzzle{
		block:      &raw,
		startNonce: *big.NewInt(0),
		target:     difficulty,
	}
	return puzzle
}

func (node *Miner) Start(mineMaxBlocks int64) {
	node.mutex.Lock()
	if node.IsRunning {
		minerLog.Printf("Miner already running")
		return
	}
	node.IsRunning = true
	node.mutex.Unlock()

	// The next tip channel.
	// next_tip := make(chan Block)
	// block_solutions := make(chan Block)
	hashrateChannel := make(chan float64, 1)
	puzzleChannel := make(chan POWPuzzle, 1)
	solutionChannel := make(chan POWPuzzle, 1)

	go MineWithStatus(hashrateChannel, solutionChannel, puzzleChannel)

	var blocksMined int64 = 0

	puzzleChannel <- node.MakeNewPuzzle()
	for {
		select {
		case hashrate := <-hashrateChannel:
			// Print iterations using commas.
			p := message.NewPrinter(language.English)
			minerLog.Printf(p.Sprintf("Hashrate: %.2f H/s\n", hashrate))
		case puzzle := <-solutionChannel:
			minerLog.Println("Received solution")

			raw := puzzle.block
			solution := puzzle.solution
			raw.SetNonce(solution)

			minerLog.Printf("Solution: hash=%s nonce=%s\n", Bytes32ToString(raw.Hash()), solution.String())

			if node.OnBlockSolution != nil {
				node.OnBlockSolution(*raw)
			}

			blocksMined += 1
			if mineMaxBlocks != -1 && mineMaxBlocks <= blocksMined {
				minerLog.Println("Mined max blocks; stopping miner")
				node.mutex.Lock()
				node.IsRunning = false
				node.mutex.Unlock()
				return
			}

			minerLog.Println("Making new puzzle")
			minerLog.Println("New puzzle ready")
			puzzleChannel <- node.MakeNewPuzzle()
		}
	}
}
