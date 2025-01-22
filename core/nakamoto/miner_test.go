package nakamoto

import (
	"database/sql"
	"testing"

	"github.com/liamzebedee/tinychain-go/core"
)

func newBlockdagForMiner() (BlockDAG, ConsensusConfig, *sql.DB) {
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

	conf := ConsensusConfig{
		EpochLengthBlocks:       5,
		TargetEpochLengthMillis: 1000,
		GenesisDifficulty:       HexStringToBigInt("0fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"),
		// https://serhack.me/articles/story-behind-alternative-genesis-block-bitcoin/ ;)
		GenesisParentBlockHash: HexStringToBytes32("000006b15d1327d67e971d1de9116bd60a3a01556c91b6ebaa416ebc0cfaa647"),
		MaxBlockSizeBytes:      2 * 1024 * 1024, // 2MB
	}

	blockdag, err := NewBlockDAGFromDB(db, stateMachine, conf)
	if err != nil {
		panic(err)
	}

	return blockdag, conf, db
}

func TestMiner(t *testing.T) {
	dag, _, _ := newBlockdagForMiner()
	minerWallet, err := core.CreateRandomWallet()
	if err != nil {
		t.Fatalf("Failed to create miner wallet: %s", err)
	}

	miner := NewMiner(dag, minerWallet)
	miner.Start(10)
}
