package nakamoto

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
)

func newStateDB() *sql.DB {
	// See: https://stackoverflow.com/questions/77134000/intermittent-table-missing-error-in-sqlite-memory-database
	useMemoryDB := true
	var connectionString string
	if useMemoryDB {
		connectionString = "file:state_db.test?mode=memory"
	} else {
		connectionString = "state_db.test.sqlite3"
	}

	db, err := OpenDB(connectionString)
	if err != nil {
		panic(err)
	}
	if useMemoryDB {
		db.SetMaxOpenConns(1)
	}
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
		RawTransaction: MakeTransferTx([65]byte{}, wallets[0].PubkeyBytes(), 100, &wallets[0], 0),
		IsCoinbase:     true,
	}
	stateMachine.Transition(tx0)

	// Assert balance.
	balance0 := stateMachine.GetBalance(wallets[0].PubkeyBytes())
	assert.Equal(t, uint64(100), balance0)

	// Now transfer coins to another account.
	tx1 := CoinStateMachineInput{
		RawTransaction: MakeTransferTx(wallets[0].PubkeyBytes(), wallets[1].PubkeyBytes(), 50, &wallets[0], 0),
		IsCoinbase:     false,
	}
	stateMachine.Transition(tx1)

	// Assert balance.
	balance1 := stateMachine.GetBalance(wallets[0].PubkeyBytes())
	assert.Equal(t, uint64(0), balance1)

	balance2 := stateMachine.GetBalance(wallets[1].PubkeyBytes())
	assert.Equal(t, uint64(100), balance2)
}

func TestNodeReorgStateMachine(t *testing.T) {
	// The state machine is always updated after a new tip is found.
	// We loop all the block txs and apply them.
	// In order to make reorgs efficient to process, we save state snapshots.
	// A state snapshot table row is simply (blockhash, txid, snapshot_id, account, balance)
	// We load the block dag and traverse the heaviest chain forwards from genesis, accumulating state snapshots and applying them to recreate the state.
	// If we need to revert the state, we can simply map the state snapshots for a blockhash and lookup the previous snapshot_id for each state leaf.
	// This is a custom state diff. The state leaves may be stored differently on disk (ie. 0 balance accounts deleted).

	// It is possible to garbage collect all state snapshots for a blockhash if the block is not in the heaviest chain.
	// It is possible to garbage collect all state snapshots except the highest snapshot_id for a unique account (leaf id). This makes it impossible to revert but saves space.

	// Create a state machine.
	// Insert some transactions.
	
}
