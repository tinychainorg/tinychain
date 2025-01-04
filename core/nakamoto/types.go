package nakamoto

import (
	"encoding/hex"
	"math/big"
	"strconv"
	"time"
)

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
// e.g. 1000_0ab1c2d3... is the epoch starting at height 1000 with start block hash 0ab1c2d3...
func (e *Epoch) GetId() string {
	return GetIdForEpoch(e.StartBlockHash, e.StartHeight)
}

type PeerConfig struct {
	ipAddress      string
	port           string
	bootstrapPeers []string
}

func NewPeerConfig(ipAddress string, port string, bootstrapPeers []string) PeerConfig {
	return PeerConfig{ipAddress: ipAddress, port: port, bootstrapPeers: bootstrapPeers}
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
	PeerId              string `json:"peerId"` // TODO temporary fix.
	// TODO add chain/network ID.
	Time time.Time `json:"time"`
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
