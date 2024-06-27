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

	// Maximum block size.
	MaxBlockSizeBytes uint64
}

// A raw block is the block as transmitted on the network.
// It contains the block header and the block body.
// It does not contain any block metadata such as height, epoch, or difficulty.
type RawBlock struct {
	// Block header.
	ParentHash [32]byte `json:"parent_hash"`
	Difficulty [32]byte `json:"difficulty"`
	Timestamp uint64    `json:"timestamp"`
	NumTransactions uint64 `json:"num_transactions"`
	TransactionsMerkleRoot [32]byte `json:"transactions_merkle_root"`
	Nonce [32]byte `json:"nonce"`
	
	// Block body.
	Transactions []RawTransaction `json:"transactions"`
}

func (b *RawBlock) HashStr() (string) {
	sl := b.Hash()
	return hex.EncodeToString(sl[:])
}

type RawTransaction struct {
	Sig [64]byte 		`json:"sig"`
	FromPubkey [65]byte `json:"from_pubkey"`
	Data []byte 		`json:"data"`
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
	Epoch string
	Work big.Int
	SizeBytes uint64
	Hash [32]byte
	AccumulatedWork big.Int
}

func (b *Block) HashStr() (string) {
	sl := b.Hash[:]
	return hex.EncodeToString(sl)
}

type Transaction struct {
	Sig [64]byte
	FromPubkey [65]byte
	Data []byte
	Hash [32]byte
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
	GetCurrentTip() (Block, error)
}

// The block DAG is the core data structure of the Nakamoto consensus protocol.
// It is a directed acyclic graph of blocks, where each block has a parent block.
// As it is infeasible to store the entirety of the blockchain in-memory, 
// the block DAG is backed by a SQL database.
type BlockDAG struct {
	// The backing SQL database store, which stores:
	// - blocks
	// - epochs
	// - transactions
	db *sql.DB

	// The state machine.
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

	// Epoch unique ID.
	Id string

	// Start block.
	StartBlockHash [32]byte
	// Start time.
	StartTime uint64
	// Start height.
	StartHeight uint64

	// Difficulty target.
	Difficulty big.Int
}

func GetIdForEpoch(startBlockHash [32]byte, startHeight uint64) (string) {
	return strconv.FormatUint(uint64(startHeight), 10) + "_" + hex.EncodeToString(startBlockHash[:])
}

// The epoch unique ID is the height ++ startblockhash.
func (e *Epoch) GetId() (string) {
	return GetIdForEpoch(e.StartBlockHash, e.StartHeight)
}

type PeerConfig struct {
	port string
    bootstrapPeers []string
}