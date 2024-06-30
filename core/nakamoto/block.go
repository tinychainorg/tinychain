// Nakamoto consensus.

package nakamoto

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
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
