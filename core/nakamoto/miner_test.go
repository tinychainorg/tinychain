package nakamoto

import (
	"testing"
	"github.com/liamzebedee/tinychain-go/core"
	"math/big"
	"database/sql"
	"encoding/hex"
)


func newBlockdagForMiner() (BlockDAG, ConsensusConfig, *sql.DB) {
	useMemoryDB := true
	var connectionString string
	if useMemoryDB {
		connectionString = "file:memdb1?mode=memory"
	} else {
		connectionString = "test.sqlite3"
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
		EpochLengthBlocks: 5,
		TargetEpochLengthMillis: 12000,
		GenesisDifficulty: *genesis_difficulty,
		GenesisBlockHash: genesisBlockHash,
		MaxBlockSizeBytes: 2*1024*1024, // 2MB
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

	miner := NewNode(dag, minerWallet)
	miner.Start()
}