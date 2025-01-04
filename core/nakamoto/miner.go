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
type Miner struct {
	dag            BlockDAG
	CoinbaseWallet *core.Wallet
	IsRunning      bool
	GraffitiTag    [32]byte

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

func (miner *Miner) MineWithStatus(hashrateChannel chan float64, solutionChannel chan POWPuzzle, puzzleChannel chan POWPuzzle) (big.Int, error) {
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
		miner.log.Println("Waiting for new puzzle")
		puzzle := <-puzzleChannel
		block := puzzle.block
		nonce := puzzle.startNonce
		target := puzzle.target
		miner.log.Printf("New puzzle block=%s target=%s\n", block.HashStr(), target.String())

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
				miner.log.Printf("Puzzle solved: iterations=%d\n", i)

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
				miner.log.Printf("New puzzle block=%s target=%s\n", block.HashStr(), target.String())
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
	raw.TransactionsMerkleRoot = GetMerkleRootForTxs(raw.Transactions)

	// Mine the POW solution.
	curr_height := current_tip.Height + 1

	// First get the right epoch.
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

func (miner *Miner) Start(mineMaxBlocks int64) []RawBlock {
	miner.mutex.Lock()
	if miner.IsRunning {
		miner.log.Printf("Miner already running")
		return []RawBlock{}
	}
	miner.IsRunning = true
	miner.mutex.Unlock()

	// The next tip channel.
	// next_tip := make(chan Block)
	// block_solutions := make(chan Block)
	hashrateChannel := make(chan float64, 1)
	puzzleChannel := make(chan POWPuzzle, 1)
	solutionChannel := make(chan POWPuzzle, 1)

	go miner.MineWithStatus(hashrateChannel, solutionChannel, puzzleChannel)

	var blocksMined int64 = 0
	mined := []RawBlock{}

	puzzleChannel <- miner.MakeNewPuzzle()
	for {
		select {
		case hashrate := <-hashrateChannel:
			// Print iterations using commas.
			p := message.NewPrinter(language.English)
			miner.log.Printf(p.Sprintf("Hashrate: %.2f H/s\n", hashrate))
		case puzzle := <-solutionChannel:
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
			puzzleChannel <- miner.MakeNewPuzzle()
		}
	}
}
