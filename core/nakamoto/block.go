package nakamoto

// Nakamoto consensus.

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"math/big"
)

type BlockHeader struct {
	ParentHash             [32]byte
	ParentTotalWork        [32]byte 
	Difficulty             [32]byte
	Timestamp              uint64
	NumTransactions        uint64
	TransactionsMerkleRoot [32]byte
	Nonce                  [32]byte
	Graffiti               [32]byte
}

type Block struct {
	// Block header.
	ParentHash             [32]byte
	ParentTotalWork        big.Int
	Difficulty             [32]byte
	Timestamp              uint64
	NumTransactions        uint64
	TransactionsMerkleRoot [32]byte
	Nonce                  [32]byte
	Graffiti               [32]byte

	// Block body.
	Transactions []RawTransaction

	// Metadata.
	Height          uint64
	Epoch           string
	Work            big.Int
	SizeBytes       uint64
	Hash            [32]byte
	AccumulatedWork big.Int
}


// A raw block is the block as transmitted on the network.
// It contains the block header and the block body.
// It does not contain any block metadata such as height, epoch, or difficulty.
type RawBlock struct {
	// Block header.
	ParentHash             [32]byte `json:"parent_hash"`
	ParentTotalWork        [32]byte `json:"parent_total_work"`
	Difficulty             [32]byte `json:"difficulty"`
	Timestamp              uint64   `json:"timestamp"`
	NumTransactions        uint64   `json:"num_transactions"`
	TransactionsMerkleRoot [32]byte `json:"transactions_merkle_root"`
	Nonce                  [32]byte `json:"nonce"`
	Graffiti               [32]byte `json:"graffiti"`

	// Block body.
	Transactions []RawTransaction `json:"transactions"`
}


// Block.
// =====================================================================================================================

func (b *Block) HashStr() string {
	sl := b.Hash[:]
	return hex.EncodeToString(sl)
}

// Convert a block to a raw block.
func (b *Block) ToRawBlock() RawBlock {
	return RawBlock{
		ParentHash:             b.ParentHash,
		ParentTotalWork:        BigIntToBytes32(b.ParentTotalWork),
		Difficulty:             BigIntToBytes32(b.Work),
		Timestamp:              b.Timestamp,
		NumTransactions:        b.NumTransactions,
		TransactionsMerkleRoot: b.TransactionsMerkleRoot,
		Nonce:                  b.Nonce,
		Graffiti:               b.Graffiti,
		Transactions:           b.Transactions,
	}
}

// RawBlock.
// =====================================================================================================================

func (b *RawBlock) SetNonce(i big.Int) {
	b.Nonce = BigIntToBytes32(i)
}

func (b *RawBlock) Bytes() []byte {
	// Encode canonically.
	buf := new(bytes.Buffer)

	err := binary.Write(buf, binary.BigEndian, b.ParentHash)
	if err != nil {
		panic(err)
	}
	err = binary.Write(buf, binary.BigEndian, b.ParentTotalWork)
	if err != nil {
		panic(err)
	}
	err = binary.Write(buf, binary.BigEndian, b.Difficulty)
	if err != nil {
		panic(err)
	}
	err = binary.Write(buf, binary.BigEndian, b.Timestamp)
	if err != nil {
		panic(err)
	}
	err = binary.Write(buf, binary.BigEndian, b.NumTransactions)
	if err != nil {
		panic(err)
	}
	err = binary.Write(buf, binary.BigEndian, b.TransactionsMerkleRoot)
	if err != nil {
		panic(err)
	}
	err = binary.Write(buf, binary.BigEndian, b.Nonce)
	if err != nil {
		panic(err)
	}
	err = binary.Write(buf, binary.BigEndian, b.Graffiti)
	if err != nil {
		panic(err)
	}

	// Encode transactions.
	for _, tx := range b.Transactions {
		err = binary.Write(buf, binary.BigEndian, tx.Bytes())
		if err != nil {
			panic(err)
		}
	}

	return buf.Bytes()
}

// Returns the envelope used for block hashing, which merklizes the transactions list into a merkle root.
func (b *RawBlock) Envelope() []byte {
	// Encode canonically.
	buf := new(bytes.Buffer)

	err := binary.Write(buf, binary.BigEndian, b.ParentHash)
	if err != nil {
		panic(err)
	}
	err = binary.Write(buf, binary.BigEndian, b.ParentTotalWork)
	if err != nil {
		panic(err)
	}
	err = binary.Write(buf, binary.BigEndian, b.Difficulty)
	if err != nil {
		panic(err)
	}
	err = binary.Write(buf, binary.BigEndian, b.Timestamp)
	if err != nil {
		panic(err)
	}
	err = binary.Write(buf, binary.BigEndian, b.NumTransactions)
	if err != nil {
		panic(err)
	}
	err = binary.Write(buf, binary.BigEndian, b.TransactionsMerkleRoot)
	if err != nil {
		panic(err)
	}
	err = binary.Write(buf, binary.BigEndian, b.Nonce)
	if err != nil {
		panic(err)
	}
	err = binary.Write(buf, binary.BigEndian, b.Graffiti)
	if err != nil {
		panic(err)
	}

	return buf.Bytes()
}

func (b *RawBlock) Hash() [32]byte {
	return sha256.Sum256(b.Envelope())
}

func (b *RawBlock) HashStr() string {
	sl := b.Hash()
	return hex.EncodeToString(sl[:])
}

func (b *RawBlock) SizeBytes() uint64 {
	// Calculate the size of the block.
	return uint64(len(b.Bytes()))
}

// BlockHeader.
// =====================================================================================================================

func (b *BlockHeader) Bytes() []byte {
	// Encode canonically.
	buf := new(bytes.Buffer)

	err := binary.Write(buf, binary.BigEndian, b.ParentHash)
	if err != nil {
		panic(err)
	}
	err = binary.Write(buf, binary.BigEndian, b.ParentTotalWork)
	if err != nil {
		panic(err)
	}
	err = binary.Write(buf, binary.BigEndian, b.Difficulty)
	if err != nil {
		panic(err)
	}
	err = binary.Write(buf, binary.BigEndian, b.Timestamp)
	if err != nil {
		panic(err)
	}
	err = binary.Write(buf, binary.BigEndian, b.NumTransactions)
	if err != nil {
		panic(err)
	}
	err = binary.Write(buf, binary.BigEndian, b.TransactionsMerkleRoot)
	if err != nil {
		panic(err)
	}
	err = binary.Write(buf, binary.BigEndian, b.Nonce)
	if err != nil {
		panic(err)
	}
	err = binary.Write(buf, binary.BigEndian, b.Graffiti)
	if err != nil {
		panic(err)
	}

	return buf.Bytes()
}

func (b *BlockHeader) BlockHash() [32]byte {
	return sha256.Sum256(b.Bytes())
}

func (b *BlockHeader) BlockHashStr() string {
	sl := b.BlockHash()
	return hex.EncodeToString(sl[:])
}
