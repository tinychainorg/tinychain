package nakamoto

import (
	"database/sql"
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/liamzebedee/tinychain-go/core"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
)

func newStateDB() *sql.DB {
	// See: https://stackoverflow.com/questions/77134000/intermittent-table-missing-error-in-sqlite-memory-database
	db, err := OpenDB(":memory:")
	if err != nil {
		panic(err)
	}
	db.SetMaxOpenConns(1)
	_, err = db.Exec("PRAGMA journal_mode = WAL;")
	if err != nil {
		panic(err)
	}

	return db
}

func TestStateMachineIdea(t *testing.T) {
	// Basically the state machine works as so:
	// - we deliver a rawtransaction
	// - we call statemachine.transition
	// - there are state snapshots
	// a state snapshot is simply the full state of the system at a block
	// - there are state diffs
	// a state diff is the difference between two state snapshots
	// how do we compute a state diff between two state snapshots?
	// - what is state?
	// (account, balance) pairs
	// a state diff is simply a set of effects we apply to get the new state
	// rather than manually engineer this, we can compute manual state diffs using diff or something similar.
	// iterate over the state namespaces:
	// - state_accounts -> hash(leaf) -> hash(account ++ balance)
	// iterate over all of the leaves, and compute the diff:
	// - additions
	// - deletions
	// maybe the state is more like:
	// create table state_accounts (account text, balance int)
	// StateAccountLeaf { Account string, Balance int }
	// (leaf StateAccountLeaf) Bytes() []byte { ... }
	//

	db := newStateDB()
	wallets := getTestingWallets(t)
	stateMachine, err := NewStateMachine(db)
	if err != nil {
		t.Fatal(err)
	}

	// Assert balance.
	balance00 := stateMachine.GetBalance(wallets[0].PubkeyBytes())
	assert.Equal(t, uint64(0), balance00)

	// Assert balances.
	// Ingest some transactions and calculate the state.
	tx0 := StateMachineInput{
		RawTransaction: MakeTransferTx(wallets[0].PubkeyBytes(), wallets[0].PubkeyBytes(), 100, 0, &wallets[0]),
		IsCoinbase:     true,
		MinerPubkey:    [65]byte{},
		BlockReward:    100,
	}
	effects, err := stateMachine.Transition(tx0)
	if err != nil {
		t.Fatal(err)
	}
	stateMachine.Apply(effects)

	// Assert balance.
	balance0 := stateMachine.GetBalance(wallets[0].PubkeyBytes())
	assert.Equal(t, uint64(100), balance0)

	// Now transfer coins to another account.
	tx1 := StateMachineInput{
		RawTransaction: MakeTransferTx(wallets[0].PubkeyBytes(), wallets[1].PubkeyBytes(), 50, 0, &wallets[0]),
		IsCoinbase:     false,
		MinerPubkey:    [65]byte{},
		BlockReward:    0,
	}
	effects, err = stateMachine.Transition(tx1)
	if err != nil {
		t.Fatal(err)
	}
	stateMachine.Apply(effects)

	// Assert balance.
	balance1 := stateMachine.GetBalance(wallets[0].PubkeyBytes())
	assert.Equal(t, uint64(50), balance1)

	balance2 := stateMachine.GetBalance(wallets[1].PubkeyBytes())
	assert.Equal(t, uint64(50), balance2)
}

func TestNodeReorgStateMachine(t *testing.T) {
	// The state machine is always updated after a new tip is found.
	// We loop all the block txs and apply them.
	// In order to make reorgs efficient to process, we save state snapshots.
	// A state snapshot table row is simply (blockhash, txid, snapshot_id, account, balance)
	// We load the block dag and traverse the heaviest chain forwards from genesis, accumulating state snapshots and applying them to recreate the state.
	// If we need to revert the state, we can simply map the state snapshots for a blockhash and lookup the previous snapshot_id for each state leaf.
	// This is a custom state diff. The state leaves may be stored differently on disk (	ie. 0 balance accounts deleted).

	// It is possible to garbage collect all state snapshots for a blockhash if the block is not in the heaviest chain.
	// It is possible to garbage collect all state snapshots except the highest snapshot_id for a unique account (leaf id). This makes it impossible to revert but saves space.

	// Create a state machine.
	// Insert some transactions.

	/////////////// A state snapshot table row is simply (blockhash, txid, snapshot_id, account, balance)

	// What happens when we reorg?
	// We have a new tip
	// We have a state machine that corresponds with a tip.
	// We can rewinding the state machine by applying state modifications in reverse.
	// Or we can just recompute the state machine from the ancestor.
	//
	// when do we need to access state anyways?
	// 1. when we're computing new state leaves / transition the machine
	// 2. when we're calling the api
	// we actually want to query the api as follows:
	// block_with_confirmations(n_confirmations) : tip.height - n_confirmations
	// GetBalanceAt(block, account) -> get state leaf
	// How do we express the state fully declaratively without being smart or efficient?
	// Transitive rule: if a leaf has not been modified since block N, then it retains the same value.
	// ie. if we are at block height 10, and the leaf was inserted (height=6, key=x, value=y), then we can simply sort by height desc and pick the latest value
	// the problem is that the height does not determine consensus - ie. you can have a chain with a deeper height but is not the most accumulated work chain
	// the real block height is actually the accWork field. This is the true timestamp?
	// (blockhash, txindex, account, balance) <- 1st order design
	// how to reconstruct the state for an arbitrary block H?
	// 1. get the full chain of block hashes (the longest chain) by following the current tip backwards
	// 2. begin at block 0, reconstruct the state:
	//   2a. block_leaves = []. foreach tx, reduce(block_leaves, statemachine.transition(tx))
	//   2b. apply all block leaves to current state {}
	//   2c. continue for next block
	// 3. At the end of this, we will have the current state.
	// Okay so this would obviously be horribly inefficient to do for every new tip we receive.
	// Where do we add caching? (ie. saving our work). And how?
	// - save the new leaves for each block
	// - save the full accumulated/reduced state for each block
	// - save the chain of block hashes for longest chain regularly (so we only do like O(32) lookups in SQL)
	// Let's say we have the longest chain of hashes. Call it "longestchainHashList"
	// And let's say we have every state leaf annotated with the block in which it was created
	// Then to get the latest value for a key, we can essentially ask - select * from state_leaves where leaf.blockhash
	// UGH how do we find the unique state leaf
	//
	// Imagine one leaf with multiple values corresponding to multiple tips:
	// - (key=1, value=50,  t=5)
	// - (key=1, value=350, t=5)
	// - (key=1, value=30,  t=5)
	//
	// - (key=1, value=50,  t=5, acc_work=100)
	// - (key=1, value=350, t=5, acc_work=80)
	// - (key=1, value=30,  t=5, acc_work=200)
	//
	// We obviously pick the key with the highest acc work right? Because this represents the longest chain.
	// The PROBLEM is that this acc_work value is probably set at the time of insertion, which means it would not be up-to-date.
	// Like for example, if someone then mines a divergent branch of the chain, produces a lot of work and updates (key=1), it might appear this key has more work than it really does?
	// No that's impossible. Because that means their chain would become the longest. We have accumulated/cumulative work for a reason.
	// No it is possible - since acc_work only refers to the work at that block in time.
	//
	// state_leafs:
	// - (key=1, value=50,  t=5, blockhash=)
	// - (key=1, value=350, t=5, blockhash=)
	// - (key=1, value=30,  t=5, blockhash=)
	//
	// state_snapshot(block) ->
	// -
	//
	// state_snapshots:
	// - (blockhash, ((key, value)))
	//

	// GetLongestChainHashList (can we make SQL index this? Maybe.)
}

// Notes on State Storage.
//
// Assuming your block size is 1mB
// Transaction size = 155 bytes
// Max number of transactions per block = 1000*1000 / 155 bytes = 6541
// Each transaction produces a new state leaf.
// State leaf size = 65 bytes (account) + 8 bytes (balance uint64) = 73 bytes
// Maximum state change per block = 6541 txs * 73 bytes = 477493 bytes = 0.47mB
// Block rate = 1 block / 10 mins
// 1 * 6 * 24 = 144 blocks / day
// 1440*0.47 = 67.68mB / day
// Total storage cost of storing snapshots for every block = 67.68mB / day
//
// Checking this math:
// - daily ethereum state growth
//   https://etherscan.io/chartsync/chaindefault
//   https://ycharts.com/indicators/ethereum_chain_full_sync_data_size
//   https://www.paradigm.xyz/2024/03/how-to-raise-the-gas-limit-1
//
// - daily bitcoin state growth
//   https://ycharts.com/indicators/bitcoin_blockchain_size
//   https://www.blockchain.com/explorer/charts/n-transactions-per-block
//   300mb/day
//
// Given that we store the full pubkey (65 bytes), and not a hash (32 bytes) or a compressed version (33 bytes), this makes sense.
// Bitcoin achieves a 50% reduction in state size by using the pubkeyhash, and its growth rate is roughly half ours.
//
// How does one prune the state then?
// 1. Rollup state older than an "unlikely to revert" threshold (6 blocks max)
// 2. ZK prove state, which is also a rollup.
// 3. Bulletproofs
//
// What about we consider space-time tradeoffs?
// How quickly can we process transactions
//
// (base) ➜  nakamoto git:(state-machine) ✗ go test -v -bench=. -run TestBenchmarkTxOpsPerDay
// === RUN   TestBenchmarkTxOpsPerDay
// Database version: 0
// Running migration: 0
// Database upgraded to: 1
//     state_machine_test.go:226: Beginning benchmark
//     state_machine_test.go:230: Processing 6541 transactions (no parallelism)
//     state_machine_test.go:237: Processed 0 transactions
//     state_machine_test.go:237: Processed 1000 transactions
//     state_machine_test.go:237: Processed 2000 transactions
//     state_machine_test.go:237: Processed 3000 transactions
//     state_machine_test.go:237: Processed 4000 transactions
//     state_machine_test.go:237: Processed 5000 transactions
//     state_machine_test.go:237: Processed 6000 transactions
// --- PASS: TestBenchmarkTxOpsPerDay (0.25s)
// That's with signatures being created.
// Without signatures:
// --- PASS: TestBenchmarkTxOpsPerDay (0.01s) - 25x speedup
//
//     state_machine_test.go:279: Processed 9419000 transactions
// --- PASS: TestBenchmarkTxOpsPerDay (9.38s)
//
//     state_machine_test.go:292: Processed 9419000 transactions
// --- PASS: TestBenchmarkTxOpsPerDay (13.92s)
// This is on an AWS Instance with 2 vCPU's, 512MB RAM, 1GB swap.
//
// So what does this infer?
// Time:  At a block time of 10mins, block size of 1mb, and 6541 txs per block, it takes 9.38s to process a 10 days worth (6541*144=9.4M) of state leaves (ignoring SQL reads).
// Time:  9.38s   (3.2 GHz, 10 12ys1 cache, Mac M1)// Space: 676.8mB
//
// It seems like it's pretty efficient to reconstruct state, and pretty storage-heavy to store the entire history.
//
// https://gist.github.com/jboner/2841832
//
// Okay so, remaining speedups:
// - paralellization (non-contentious state writes) - probably 10x
// - disk reads - probably -2x
//
// One of the core considerations:
// There are 2 approaches to the state construction:
// 1. Space tradeoff - store 60mb a day of state snapshots.
// 2. Time tradeoff  - spend 0.96s to process a day of state transitions.
//

func newUnsignedTransferTx(from [65]byte, to [65]byte, amount uint64, wallet *core.Wallet, fee uint64) RawTransaction {
	tx := RawTransaction{
		Version:    1,
		Sig:        [64]byte{},
		FromPubkey: from,
		ToPubkey:   to,
		Amount:     amount,
		Fee:        fee,
		Nonce:      0,
	}
	return tx
}

func TestBenchmarkTxOpsPerDay(t *testing.T) {
	db := newStateDB()
	wallets := getTestingWallets(t)
	stateMachine, err := NewStateMachine(db)
	if err != nil {
		t.Fatal(err)
	}

	// Assert balance.
	balance00 := stateMachine.GetBalance(wallets[0].PubkeyBytes())
	assert.Equal(t, uint64(0), balance00)

	// Perform the following:
	// 1. Coinbase mint 100 coins
	// 2. Transfer them to another account
	t.Logf("Beginning benchmark")
	txsProcessed := 0
	maxTotalTxsPerDay := 6541 * 144
	// maxTotalTxsPerBlock := 6541
	maxTxs := maxTotalTxsPerDay
	t.Logf("Processing %d transactions (no parallelism)", maxTxs)

	// stateMachine.state[wallets[0].PubkeyBytes()] = 1000000000000

	for {
		if txsProcessed >= maxTxs {
			break
		}
		if txsProcessed%1000 == 0 {
			t.Logf("Processed %d transactions", txsProcessed)
		}

		// 1. Coinbase mint.
		coinbaseTx := StateMachineInput{
			RawTransaction: newUnsignedTransferTx(wallets[0].PubkeyBytes(), wallets[0].PubkeyBytes(), 100, &wallets[0], 0),
			IsCoinbase:     true,
			MinerPubkey:    [65]byte{},
			BlockReward:    100,
		}
		effects, err := stateMachine.Transition(coinbaseTx)
		if err != nil {
			t.Fatal(err)
		}
		stateMachine.Apply(effects)
		txsProcessed += 1

		// 2. Simple transfer.
		tx1 := StateMachineInput{
			RawTransaction: newUnsignedTransferTx(wallets[0].PubkeyBytes(), wallets[1].PubkeyBytes(), 50, &wallets[0], 0),
			IsCoinbase:     false,
			MinerPubkey:    [65]byte{},
			BlockReward:    0,
		}
		effects, err = stateMachine.Transition(tx1)
		if err != nil {
			t.Fatal(err)
		}
		stateMachine.Apply(effects)
		txsProcessed += 1
	}
}

func newBlockdagForStateMachine() (BlockDAG, ConsensusConfig, *sql.DB) {
	db, err := OpenDB(":memory:")
	if err != nil {
		panic(err)
	}
	db.SetMaxOpenConns(1) // :memory: only
	_, err = db.Exec("PRAGMA journal_mode = WAL;")
	if err != nil {
		panic(err)
	}

	stateMachine := newMockStateMachine()

	genesis_difficulty := new(big.Int)
	genesis_difficulty.SetString("0fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 16)

	// https://serhack.me/articles/story-behind-alternative-genesis-block-bitcoin/ ;)
	genesisBlockHash_, err := hex.DecodeString("000006b15d1327d67e971d1de9116bd60a3a01556c91b6ebaa416ebc0cfaa646")
	if err != nil {
		panic(err)
	}
	genesisBlockHash := [32]byte{}
	copy(genesisBlockHash[:], genesisBlockHash_)

	conf := ConsensusConfig{
		EpochLengthBlocks:       5,
		TargetEpochLengthMillis: 1000,
		GenesisDifficulty:       *genesis_difficulty,
		GenesisParentBlockHash:  genesisBlockHash,
		MaxBlockSizeBytes:       2 * 1024 * 1024, // 2MB
	}

	blockdag, err := NewBlockDAGFromDB(db, stateMachine, conf)
	if err != nil {
		panic(err)
	}

	return blockdag, conf, db
}

func TestStateMachineReconstructState(t *testing.T) {
	/*

		Okay so this is how we do:

		latest_tip, err := dag.GetLatestTip()
		longestChainHashList, err := dag.GetLongestChainHashList(latest_tip.Hash, latest_tip.Height)

		state = {}
		stateMachine = StateMachine{}

		for i, blockhash := range longestChainHashList {
		  // get txs
		  txs = "select from, to, amount, fee, version from txs where blockhash = ? order by txindex", blockhash
		  // map
		  new_leaves = txs.map(tx => stateMachine.Transition(state, tx))
		  // reduce
		  new_leaves.map(leaves => stateMachine.Apply(leaves))
		}

		After all of this, the state will be up to date for the current block.
	*/

	// assert := assert.New(t)
	dag, _, _ := newBlockdagForStateMachine()
	wallets := getTestingWallets(t)
	stateMachine, err := NewStateMachine(dag.db)
	if err != nil {
		t.Fatal(err)
	}
	miner := NewMiner(dag, &wallets[0])

	// Mine 100 blocks.
	miner.OnBlockSolution = func(block RawBlock) {
		err := dag.IngestBlock(block)
		if err != nil {
			t.Fatal(err)
		}
	}
	miner.Start(100)

	// Get the latest tip.
	latestTip, err := dag.GetLatestFullTip()
	if err != nil {
		t.Fatal(err)
	}

	// Get the longest chain hash list.
	longestChainHashList, err := dag.GetLongestChainHashList(latestTip.Hash, latestTip.Height)
	if err != nil {
		t.Fatal(err)
	}

	for _, blockHash := range longestChainHashList {
		// 1. Get all transactions for block.
		// TODO ignore: nonce, sig
		txs, err := dag.GetBlockTransactions(blockHash)
		if err != nil {
			t.Fatal(err)
		}

		t.Logf("Processing block %x with %d transactions", blockHash, len(*txs))

		// 2. Map transactions to state leaves through state machine transition function.
		var stateMachineInput StateMachineInput
		var minerPubkey [65]byte
		isCoinbase := false

		for i, tx := range *txs {
			// Special case: coinbase tx is always the first tx in the block.
			if i == 0 {
				minerPubkey = tx.FromPubkey
				isCoinbase = true
			}

			// Construct the state machine input.
			stateMachineInput = StateMachineInput{
				RawTransaction: tx.ToRawTransaction(),
				IsCoinbase:     isCoinbase,
				MinerPubkey:    minerPubkey,
				BlockReward:    0,
			}

			// Transition the state machine.
			effects, err := stateMachine.Transition(stateMachineInput)
			if err != nil {
				t.Logf("Error transitioning state machine: block=%x txindex=%d error=\"%s\"", blockHash, i, err)
			}

			// Apply the effects.
			stateMachine.Apply(effects)

			if i == 0 {
				isCoinbase = false
			}
		}
	}

	// 3. The state is now reconstructed.
	// Loop through the state machine and print the balances.
	for _, wallet := range wallets {
		balance := stateMachine.GetBalance(wallet.PubkeyBytes())
		t.Logf("Account %x has balance %d", wallet.PubkeyBytes(), balance)
	}
}

func assertIntEqual[num int | uint | int8 | uint8 | int16 | uint16 | int32 | uint32 | int64 | uint64](t *testing.T, a num, b num) {
	t.Helper()
	if a != b {
		t.Errorf("Expected %d to equal %d", a, b)
	}
}

// Test the constraint that we only process a transaction once (ie. no replay attacks).
func TestStateMachineTxAlreadySequenced(t *testing.T) {
	dag, _, _ := newBlockdagForStateMachine()
	wallets := getTestingWallets(t)
	state, err := NewStateMachine(dag.db)
	if err != nil {
		t.Fatal(err)
	}
	miner := NewMiner(dag, &wallets[0])

	// Mine 10 blocks.
	miner.OnBlockSolution = func(block RawBlock) {
		err := dag.IngestBlock(block)
		if err != nil {
			t.Fatal(err)
		}
	}
	miner.Start(10)

	// Apply transactions into state.
	longestChainHashList, err := dag.GetLongestChainHashList(dag.FullTip.Hash, dag.FullTip.Height)
	if err != nil {
		t.Fatalf("Failed to get longest chain hash list: %s\n", err)
	}
	state, err = RebuildState(&dag, *state, longestChainHashList)
	if err != nil {
		t.Fatalf("Failed to rebuild state: %s\n", err)
	}

	wallet0_balance1 := state.GetBalance(wallets[0].PubkeyBytes())
	assertIntEqual(t, uint64(50*10)*ONE_COIN, wallet0_balance1) // coinbase rewards.

	// Now we send a transfer tx.
	// First create the tx, then mine a block with it.
	rawTx := MakeTransferTx(wallets[0].PubkeyBytes(), wallets[1].PubkeyBytes(), 100, 0, &wallets[0])
	miner.GetBlockBody = func() BlockBody {
		return []RawTransaction{rawTx}
	}
	// This should succeed.
	miner.Start(1)

	// Rebuild state.
	state2, err := NewStateMachine(dag.db)
	if err != nil {
		t.Fatal(err)
	}
	longestChainHashList, err = dag.GetLongestChainHashList(dag.FullTip.Hash, dag.FullTip.Height)
	if err != nil {
		t.Fatalf("Failed to get longest chain hash list: %s\n", err)
	}
	state2, err = RebuildState(&dag, *state2, longestChainHashList)
	if err != nil {
		t.Fatalf("Failed to rebuild state: %s\n", err)
	}

	// Check the transfer tx was processed.
	wallet0_balance2 := state2.GetBalance(wallets[0].PubkeyBytes())
	wallet1_balance1 := state2.GetBalance(wallets[1].PubkeyBytes())
	blockReward := GetBlockReward(int(dag.FullTip.Height))
	assertIntEqual(t, wallet0_balance1+blockReward-100, wallet0_balance2)
	assertIntEqual(t, uint64(100), wallet1_balance1)

	// Now we test transaction replay.
	// This only involves testing the state machine itself.
	replayAttackInput := StateMachineInput{
		RawTransaction: rawTx,
		IsCoinbase:     false,
		MinerPubkey:    miner.CoinbaseWallet.PubkeyBytes(),
		BlockReward:    0,
	}

	t.Skip()
	// TODO: we will finish the state machine unique tx sequence constraint later.

	effects, err := state2.Transition(replayAttackInput)
	if err == nil {
		t.Fatalf("Expected transaction to be rejected\n")
	}

	assert.Equal(t, "transaction already sequenced", err.Error())
	assertIntEqual(t, 0, len(effects))
}
