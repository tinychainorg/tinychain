// Nakamoto consensus.

package nakamoto

import (
	"bytes"
	"encoding/binary"
	"math/big"
	"crypto/sha256"
)

func (b *RawBlock) SetNonce(i big.Int) {
	b.Nonce = BigIntToBytes32(i)
}

func (b *RawBlock) Envelope() []byte {
	// Encode canonically.
	buf := new(bytes.Buffer)

	err := binary.Write(buf, binary.BigEndian, b.ParentHash)
	if err != nil { panic(err); }
	err = binary.Write(buf, binary.BigEndian, b.Timestamp)
	err = binary.Write(buf, binary.BigEndian, b.NumTransactions)
	if err != nil { panic(err); }
	err = binary.Write(buf, binary.BigEndian, b.TransactionsMerkleRoot)
	if err != nil { panic(err); }
	err = binary.Write(buf, binary.BigEndian, b.Nonce)
	if err != nil { panic(err); }

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

func (tx *RawTransaction) Envelope() []byte {
	buf := new(bytes.Buffer)

	err := binary.Write(buf, binary.BigEndian, tx.FromPubkey)
	if err != nil { panic(err); }
	err = binary.Write(buf, binary.BigEndian, tx.Data)
	if err != nil { panic(err); }

	return buf.Bytes()
}

func (tx *RawTransaction) Hash() [32]byte {
	// Hash the envelope.
	h := sha256.New()
	h.Write(tx.Envelope())
	return sha256.Sum256(h.Sum(nil))
}

