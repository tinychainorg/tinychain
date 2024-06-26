package nakamoto

import (
	"math/big"
	"strconv"
	"encoding/hex"
	"database/sql"
)

type ConsensusConfig struct {
	// The length of an epoch.
	EpochLengthBlocks uint64

	// The target block production rate in terms of 1 epoch.
	TargetEpochLengthMillis uint64

	// Genesis difficulty target.
	GenesisDifficulty big.Int

	// The genesis block hash.
	GenesisBlockHash [32]byte
}

// A raw block is the block as transmitted on the network.
// It contains the block header and the block body.
// It does not contain any block metadata such as height, epoch, or difficulty.
type RawBlock struct {
	// Block header.
	ParentHash [32]byte
	Timestamp uint64
	NumTransactions uint64
	TransactionsMerkleRoot [32]byte
	Nonce [32]byte
	
	// Block body.
	Transactions []RawTransaction
}

type RawTransaction struct {
	Sig [64]byte
	FromPubkey [64]byte
	Data []byte
}

type Block struct {
	// Block header.
	ParentHash [32]byte
	Timestamp uint64
	NumTransactions uint64
	TransactionsMerkleRoot [32]byte
	Nonce [32]byte
	
	// Block body.
	Transactions []RawTransaction

	// Metadata.
	Height uint64
	Epoch uint64
	Work big.Int
	SizeBytes uint64
}

type BlockDAGInterface interface {
	// Ingest block.
	IngestBlock(b Block) error

	// Get block.
	GetBlockByHash(hash [32]byte) (Block)

	// Get epoch for block.
	GetEpochForBlockHash(parentBlockHash [32]byte) (uint64, error)
	
	// Get a list of blocks at height.
	GetBlocksByHeight(height uint64) ([]Block, error)

	// Get the tip of the chain, given a minimum number of confirmations.
	GetTips(minConfirmations uint64) ([]Block, error)
}

type BlockDAG struct {
	// The backing SQL database store.
	db *sql.DB
	stateMachine StateMachine

	// Consensus settings.
	consensus ConsensusConfig
}

type StateMachine interface {
	VerifyTx(tx RawTransaction) error
}

type Epoch struct {
	// Epoch number.
	Number uint64

	// Start block.
	StartBlockHash [32]byte
	// Start time.
	StartTime uint64
	// Start height.
	StartHeight uint64

	// End block.
	EndBlockHash [32]byte
	// End time.
	EndTime uint64

	// Difficulty target.
	Difficulty big.Int
}

// TODO BROKEN
// The epoch unique ID is the height ++ startblockhash.
func (e *Epoch) Id() (string) {
	return strconv.FormatUint(uint64(e.StartHeight), 10) + "_" + hex.EncodeToString(e.StartBlockHash[:])
}