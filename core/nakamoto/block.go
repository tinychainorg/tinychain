// Nakamoto consensus.

package nakamoto

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"math/big"
)

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
	// Hash the envelope.
	h := sha256.New()
	h.Write(b.Envelope())
	return sha256.Sum256(h.Sum(nil))
}

func (b *RawBlock) SizeBytes() uint64 {
	// Calculate the size of the block.
	return uint64(len(b.Envelope()))
}

func (b *BlockHeader) Envelope() []byte {
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

func (b *BlockHeader) Hash() [32]byte {
	// Hash the envelope.
	h := sha256.New()
	h.Write(b.Envelope())
	return sha256.Sum256(h.Sum(nil))
}

func (b *BlockHeader) HashStr() string {
	sl := b.Hash()
	return hex.EncodeToString(sl[:])
}

func (b *RawBlock) Header() (header [208]byte) {
	// total header size = 1 + 32 + 32 + 32 + 8 + 8 + 32 + 32 + 32

	// Parent hash.
	copy(header[0:32], b.ParentHash[:])
	// Parent total work.
	copy(header[32:64], b.ParentTotalWork[:])
	// Difficulty.
	copy(header[64:96], b.Difficulty[:])
	// Timestamp.
	timestamp := make([]byte, 8)
	binary.BigEndian.PutUint64(timestamp, b.Timestamp)
	copy(header[96:104], timestamp)
	// Num transactions.
	numTransactions := make([]byte, 8)
	binary.BigEndian.PutUint64(numTransactions, b.NumTransactions)
	copy(header[104:112], numTransactions)
	// Transactions merkle root.
	copy(header[112:144], b.TransactionsMerkleRoot[:])
	// Nonce.
	copy(header[144:176], b.Nonce[:])
	// Graffiti.
	copy(header[176:208], b.Graffiti[:])

	return header
}

func (b *RawBlock) HashStr() string {
	sl := b.Hash()
	return hex.EncodeToString(sl[:])
}
