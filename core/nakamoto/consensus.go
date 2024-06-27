// Nakamoto consensus.

package nakamoto

import (
	"bytes"
	"encoding/binary"
	"math/big"
	"crypto/sha256"
	"fmt"
	"time"
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

func VerifyPOW(blockhash [32]byte, target big.Int) bool {
	fmt.Printf("VerifyPOW target: %s\n", target.String())

	hash := new(big.Int).SetBytes(blockhash[:])
	return hash.Cmp(&target) == -1
}

func SolvePOW(b RawBlock, startNonce big.Int, target big.Int, maxIterations uint64) (big.Int, error) {
	fmt.Printf("SolvePOW target: %s\n", target.String())

	block := b
	nonce := startNonce
	var i uint64 = 0

	for {
		i++
		
		// Exit if iterations is reached.
		if maxIterations < i {
			return big.Int{}, fmt.Errorf("Solution not found in %d iterations", maxIterations)
		}

		// Increment nonce.
		nonce.Add(&nonce, big.NewInt(1))
		block.SetNonce(nonce)

		// Hash.
		h := block.Hash()
		hash := new(big.Int).SetBytes(h[:])

		// Check solution: hash < target.
		if hash.Cmp(&target) == -1 {
			fmt.Printf("Solved in %d iterations\n", i)
			fmt.Printf("Hash: %x\n", hash.String())
			fmt.Printf("Nonce: %s\n", nonce.String())
			return nonce, nil
		}
	}
}

func Timestamp() uint64 {
	now := time.Now()
	milliseconds := now.UnixMilli()
	return uint64(milliseconds)
}

func BigIntToBytes32(i big.Int) (fbuf [32]byte) {
	buf := make([]byte, 32)
	i.FillBytes(buf)
	copy(fbuf[:], buf)
	return fbuf
}

func Bytes32ToBigInt(b [32]byte) big.Int {
	return *new(big.Int).SetBytes(b[:])
}