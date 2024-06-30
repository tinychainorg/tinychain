package nakamoto

import (
	"database/sql"
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
	stateMachine, err := NewCoinStateMachine(db)
	if err != nil {
		t.Fatal(err)
	}

	// Assert balance.
	balance00 := stateMachine.GetBalance(wallets[0].PubkeyBytes())
	assert.Equal(t, uint64(0), balance00)

	// Assert balances.
	// Ingest some transactions and calculate the state.
	tx0 := CoinStateMachineInput{
		RawTransaction: MakeTransferTx(wallets[0].PubkeyBytes(), wallets[0].PubkeyBytes(), 100, &wallets[0], 0),
		IsCoinbase:     true,
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
	tx1 := CoinStateMachineInput{
		RawTransaction: MakeTransferTx(wallets[0].PubkeyBytes(), wallets[1].PubkeyBytes(), 50, &wallets[0], 0),
		IsCoinbase:     false,
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
// 10 * 6 * 24 = 1440 blocks / day
// 1440*0.47 = 676.8mB / day
// Total storage cost of storing snapshots for every block = 676.8mB / day
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
// So what does this infer?
// Time:  At a block time of 10mins, block size of 1mb, and 6541 txs per block, it takes 9.38s to process a day's worth (6541*1440=9.4M) of state leaves (ignoring SQL reads).
// Time:  9.38s   (3.2 GHz, 128 L1 cache, Mac M1)
// Space: 676.8mB
// 
// It seems like it's pretty efficient to reconstruct state, and pretty storage-heavy to store the entire history.
// 
// https://gist.github.com/jboner/2841832
// 
// Okay so, remaining speedups:
// - paralellization (non-contentious state writes) - probably 10x
// 
// 

func newUnsignedTransferTx(from [65]byte, to [65]byte, amount uint64, wallet *core.Wallet, fee uint64) RawTransaction {
	tx := RawTransaction{
		Version: 1,
		Sig:        [64]byte{},
		FromPubkey: from,
		ToPubkey:     to,
		Amount: amount,
		Fee:    fee,
		Nonce:  0,
	}
	return tx
}

func TestBenchmarkTxOpsPerDay(t *testing.T) {
	db := newStateDB()
	wallets := getTestingWallets(t)
	stateMachine, err := NewCoinStateMachine(db)
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
	maxTotalTxsPerDay := 6541 * 1440
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
		coinbaseTx := CoinStateMachineInput{
			RawTransaction: newUnsignedTransferTx(wallets[0].PubkeyBytes(), wallets[0].PubkeyBytes(), 100, &wallets[0], 0),
			IsCoinbase:     true,
		}
		effects, err := stateMachine.Transition(coinbaseTx)
		if err != nil {
			t.Fatal(err)
		}
		stateMachine.Apply(effects)
		txsProcessed += 1

		// 2. Simple transfer.
		tx1 := CoinStateMachineInput{
			RawTransaction: newUnsignedTransferTx(wallets[0].PubkeyBytes(), wallets[1].PubkeyBytes(), 50, &wallets[0], 0),
			IsCoinbase:     false,
		}
		effects, err = stateMachine.Transition(tx1)
		if err != nil {
			t.Fatal(err)
		}
		stateMachine.Apply(effects)
		txsProcessed += 1
	}

	
}