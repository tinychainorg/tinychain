package nakamoto

import (
	"log"
	"math/big"
	"time"

	"sync"

	"github.com/liamzebedee/tinychain-go/core"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// The Miner is responsible for solving the Hashcash proof-of-work puzzle.
//
// The operation of the miner is as follows:
// 1. Begin the miner thread.
// 2. Generate a new POW puzzle:
//   a. Create the block template.
//      i. Get the current tip and set block.parent_hash to the tip's hash.
//      ii. Construct the coinbase transaction with our miner wallet.
//      iii. Get the block's body (transactions) using `Miner.GetBlockBody`. This is used to connect the mempool.
//      iv. Compute the transaction merkle root.
//   b. Compute the difficulty target for mining.
// 3. Begin mining the puzzle:
//   a. Send the puzzle to the miner thread.
//   b. The miner thread will mine the puzzle until a solution is found.
//     i. Increment the nonce.
//     ii. Hash the block.
//     iii. Check if the guess (hash) is less than the target.
//     iv. If the guess is less than the target, the puzzle is solved. Send the solution back to the main thread.
//     v. If a new puzzle is received, stop mining the current puzzle and start mining the new puzzle.
// 4. When a solution to the puzzle is found, the miner thread will send the solution back to the main thread.
// 5. The main thread will:
//   a. Set the nonce in the block to the solution.
//   b. Call `Miner.OnBlockSolution` with the block.
//   c. Increment the number of blocks mined.
//   d. If the maximum number of blocks to mine has been reached, stop the miner.
//   e. Otherwise, generate a new puzzle and send it to the miner thread.
//

type Miner struct {
	dag            BlockDAG
	CoinbaseWallet *core.Wallet
	IsRunning      bool
	GraffitiTag    [32]byte

	// Miner state.
	stopCh  chan bool
	puzzles chan POWPuzzle

	// Mutex.
	mutex sync.Mutex

	// OnBlockSolution is called when a block's hashcash puzzle is solved.
	OnBlockSolution func(block RawBlock)

	// GetTipForMining is an optional callback that can be used to override the tip used for mining.
	GetTipForMining func() Block

	// GetBlockBody is an optional callback that can be used to override the block body used for mining.
	// By default, the miner constructs a block with just a coinbase transaction.
	GetBlockBody func() BlockBody

	log *log.Logger
}

func NewMiner(dag BlockDAG, coinbaseWallet *core.Wallet) *Miner {
	return &Miner{
		dag:            dag,
		CoinbaseWallet: coinbaseWallet,
		IsRunning:      false,
		mutex:          sync.Mutex{},
		log:            NewLogger("miner", ""),
	}
}

func MakeCoinbaseTx(wallet *core.Wallet, amount uint64) RawTransaction {
	// Construct coinbase tx.
	tx := RawTransaction{
		Version:    1,
		Sig:        [64]byte{},
		FromPubkey: wallet.PubkeyBytes(),
		ToPubkey:   wallet.PubkeyBytes(),
		Amount:     amount,
		Fee:        0,
		Nonce:      randomNonce(),
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

// Miner thread:
// States:
// - waiting for puzzle
// - mining puzzle
// - puzzle solved
// - restart on new puzzle
func mineWithStatus(log *log.Logger, hashrates chan float64, solutions chan POWPuzzle, puzzles chan POWPuzzle, stopCh chan bool) (big.Int, error) {
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
			hashrates <- hashrate
			numHashes = 0
			lastHashrateMeasurement = now
		}
	})()

	// Routine: Mine.
	for {
		var i uint64 = 0
		log.Println("Waiting for new puzzle")
		puzzle := <-puzzles
		block := puzzle.block
		nonce := puzzle.startNonce
		target := puzzle.target
		log.Printf("New puzzle block=%s target=%s\n", block.HashStr(), target.String())

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

			// miner.log.Printf("hash block=%s i=%d\n", Bytes32ToString(h), i)

			// Check solution: hash < target.
			if guess.Cmp(&target) == -1 {
				log.Printf("Puzzle solved: iterations=%d\n", i)

				puzzle.solution = nonce
				solutions <- puzzle
				break
			}

			// Check if new puzzle has been received.
			select {
			case newPuzzle := <-puzzles:
				puzzle = newPuzzle
				block = puzzle.block
				nonce = puzzle.startNonce
				target = puzzle.target
				log.Printf("New puzzle block=%s target=%s\n", block.HashStr(), target.String())
			case <-stopCh:
				log.Println("Stopping miner")
				return nonce, nil
			default:
				// Do nothing.
			}
		}
	}
}

// Creates a new block template for mining.
func (miner *Miner) MakeNewPuzzle() POWPuzzle {
	// Get the current tip.
	current_tip, err := miner.dag.GetLatestFullTip()
	if err != nil {
		miner.log.Printf("Failed to get current tip: %s", err)
		panic(err)
	}
	if miner.GetTipForMining != nil {
		current_tip = miner.GetTipForMining()
	}

	// Construct coinbase tx.
	blockReward := GetBlockReward(int(current_tip.Height))
	coinbaseTx := MakeCoinbaseTx(miner.CoinbaseWallet, blockReward)

	// Get the block body.
	blockBody := []RawTransaction{}
	blockBody = append(blockBody, coinbaseTx)
	if miner.GetBlockBody != nil {
		miner.log.Printf("Getting block body for mining")
		blockBody = append(blockBody, miner.GetBlockBody()...)
	}

	// Construct block template for mining.
	raw := RawBlock{
		ParentHash:             current_tip.Hash,
		ParentTotalWork:        BigIntToBytes32(current_tip.AccumulatedWork),
		Timestamp:              Timestamp(),
		NumTransactions:        uint64(len(blockBody)),
		TransactionsMerkleRoot: [32]byte{},
		Nonce:                  [32]byte{},
		Transactions:           blockBody,
		Graffiti:               miner.GraffitiTag,
	}

	// Compute the transaction merkle root.
	raw.TransactionsMerkleRoot = GetMerkleRootForTxs(raw.Transactions)

	// Compute the difficulty target for mining, which may involve recomputing the difficulty (epoch change).
	curr_height := current_tip.Height + 1
	var difficulty big.Int
	epoch, err := miner.dag.GetEpochForBlockHash(current_tip.Hash)
	if err != nil {
		miner.log.Printf("Failed to get epoch for block hash: %s", err)
		panic(err)
	}
	if curr_height%miner.dag.consensus.EpochLengthBlocks == 0 {
		difficulty = RecomputeDifficulty(epoch.StartTime, raw.Timestamp, epoch.Difficulty, miner.dag.consensus.TargetEpochLengthMillis, miner.dag.consensus.EpochLengthBlocks, curr_height)
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

func (miner *Miner) Stop() {
	miner.mutex.Lock()
	if !miner.IsRunning {
		miner.log.Printf("Miner not running")
		miner.mutex.Unlock()
		return
	}
	miner.IsRunning = false
	miner.mutex.Unlock()

	miner.log.Println("Sent stop signal to miner")
	miner.stopCh <- true
}

// Send new puzzle to miner thread, based off the latest full tip.
func (miner *Miner) RestartWithNewPuzzle() {
	miner.puzzles <- miner.MakeNewPuzzle()
}

func (miner *Miner) Start(mineMaxBlocks int64) []RawBlock {
	miner.mutex.Lock()
	if miner.IsRunning {
		miner.log.Printf("Miner already running")
		return []RawBlock{}
	}
	miner.IsRunning = true
	miner.mutex.Unlock()

	// Set miner thread state.
	hashrates := make(chan float64, 1)
	puzzles := make(chan POWPuzzle, 1)
	solutions := make(chan POWPuzzle, 1)
	stopCh := make(chan bool, 1)

	// Set miner state.
	miner.stopCh = stopCh
	miner.puzzles = puzzles

	go mineWithStatus(miner.log, hashrates, solutions, puzzles, stopCh)

	var blocksMined int64 = 0
	mined := []RawBlock{}

	puzzles <- miner.MakeNewPuzzle()
	for {
		select {
		case hashrate := <-hashrates:
			// Print iterations using commas.
			p := message.NewPrinter(language.English)
			miner.log.Printf(p.Sprintf("Hashrate: %.2f H/s\n", hashrate))
		case puzzle := <-solutions:
			miner.log.Println("Received solution")

			raw := puzzle.block
			solution := puzzle.solution
			raw.SetNonce(solution)

			miner.log.Printf("Solution: hash=%s nonce=%s\n", Bytes32ToString(raw.Hash()), solution.String())

			if miner.OnBlockSolution != nil {
				miner.OnBlockSolution(*raw)
			}

			blocksMined += 1
			mined = append(mined, *raw)

			if mineMaxBlocks != -1 && mineMaxBlocks <= blocksMined {
				miner.log.Println("Mined max blocks; stopping miner")
				miner.mutex.Lock()
				miner.IsRunning = false
				miner.mutex.Unlock()
				return mined
			}

			miner.log.Println("Making new puzzle")
			miner.log.Println("New puzzle ready")
			puzzles <- miner.MakeNewPuzzle()
		}
	}
}
