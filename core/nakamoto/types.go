package nakamoto

import (
	"encoding/hex"
	"math/big"
	"strconv"
	"time"
)

// The Nakamoto consensus configuration, pertaining to difficulty readjustment, genesis block, and block size.
type ConsensusConfig struct {
	// The length of an epoch.
	EpochLengthBlocks uint64 `json:"epoch_length_blocks"`

	// The target block production rate in terms of 1 epoch.
	TargetEpochLengthMillis uint64 `json:"target_epoch_length_millis"`

	// Genesis difficulty target.
	GenesisDifficulty big.Int `json:"genesis_difficulty"`

	// The genesis parent block hash.
	GenesisParentBlockHash [32]byte `json:"genesis_block_hash"`

	// Maximum block size.
	MaxBlockSizeBytes uint64 `json:"max_block_size_bytes"`
}

type StateMachineInterface interface {
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

func GetIdForEpoch(startBlockHash [32]byte, startHeight uint64) string {
	return strconv.FormatUint(uint64(startHeight), 10) + "_" + hex.EncodeToString(startBlockHash[:])
}

// The epoch unique ID is the height ++ startblockhash.
func (e *Epoch) GetId() string {
	return GetIdForEpoch(e.StartBlockHash, e.StartHeight)
}

type PeerConfig struct {
	address        string
	port           string
	bootstrapPeers []string
}

func NewPeerConfig(address string, port string, bootstrapPeers []string) PeerConfig {
	return PeerConfig{address: address, port: port, bootstrapPeers: bootstrapPeers}
}

type NetworkMessage struct {
	Type string `json:"type"`
}

type HeartbeatMesage struct {
	Type                string `json:"type"` // "heartbeat"
	TipHash             string `json:"tipHash"`
	TipHeight           int    `json:"tipHeight"`
	ClientVersion       string `json:"clientVersion"`
	WireProtocolVersion uint   `json:"wireProtocolVersion"`
	ClientAddress       string `json:"clientAddress"`
	// TODO add chain/network ID.
	Time time.Time
}

// get_tip
type GetTipMessage struct {
	Type string      `json:"type"` // "get_tip"
	Tip  BlockHeader `json:"tip"`
}

// new_block
type NewBlockMessage struct {
	Type     string   `json:"type"` // "new_block"
	RawBlock RawBlock `json:"rawBlock"`
}

// new_transaction
type NewTransactionMessage struct {
	Type           string         `json:"type"` // "new_transaction"
	RawTransaction RawTransaction `json:"rawTransaction"`
}

// get_blocks
type GetBlocksMessage struct {
	Type        string   `json:"type"` // "get_blocks"
	BlockHashes []string `json:"blockHashes"`
}

type GetBlocksReply struct {
	Type          string   `json:"type"` // "get_blocks_reply"
	RawBlockDatas [][]byte `json:"rawBlockDatas"`
}

// has_block
type HasBlockMessage struct {
	Type      string `json:"type"` // "have_block"
	BlockHash string `json:"blockHash"`
}

type HasBlockReply struct {
	Type string `json:"type"` // "have_block_reply"
	Has  bool   `json:"has"`
}

// gossip_peers
type GossipPeersMessage struct {
	Type  string   `json:"type"` // "gossip_peers"
	Peers []string `json:"myPeers"`
}
