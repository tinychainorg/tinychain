package nakamoto

import (
	"crypto/sha256"
	"encoding/binary"

	"github.com/liamzebedee/tinychain-go/core"
)

func (tx *RawTransaction) Bytes() []byte {
	buf := make([]byte, 0)
	buf = append(buf, tx.Version)
	buf = append(buf, tx.Sig[:]...)
	buf = append(buf, tx.FromPubkey[:]...)
	buf = append(buf, tx.ToPubkey[:]...)

	amount := make([]byte, 8)
	binary.BigEndian.PutUint64(amount, tx.Amount)
	buf = append(buf, amount...)

	fee := make([]byte, 8)
	binary.BigEndian.PutUint64(fee, tx.Fee)
	buf = append(buf, fee...)

	nonce := make([]byte, 8)
	binary.BigEndian.PutUint64(nonce, tx.Nonce)
	buf = append(buf, nonce...)

	return buf
}

func (tx *RawTransaction) Envelope() []byte {
	buf := make([]byte, 0)
	buf = append(buf, tx.Version)
	buf = append(buf, tx.FromPubkey[:]...)
	buf = append(buf, tx.ToPubkey[:]...)

	amount := make([]byte, 8)
	binary.BigEndian.PutUint64(amount, tx.Amount)
	buf = append(buf, amount...)

	fee := make([]byte, 8)
	binary.BigEndian.PutUint64(fee, tx.Fee)
	buf = append(buf, fee...)

	nonce := make([]byte, 8)
	binary.BigEndian.PutUint64(nonce, tx.Nonce)
	buf = append(buf, nonce...)

	return buf
}

func (tx *RawTransaction) Hash() [32]byte {
	// Hash the envelope.
	h := sha256.New()
	h.Write(tx.Envelope())
	return sha256.Sum256(h.Sum(nil))
}

func MakeTransferTx(from [65]byte, to [65]byte, amount uint64, wallet *core.Wallet, fee uint64) RawTransaction {
	tx := RawTransaction{
		Version:    1,
		Sig:        [64]byte{},
		FromPubkey: from,
		ToPubkey:   to,
		Amount:     amount,
		Fee:        fee,
		Nonce:      0,
	}
	// Sign tx.
	sig, err := wallet.Sign(tx.Envelope())
	if err != nil {
		panic(err)
	}
	copy(tx.Sig[:], sig)
	return tx
}
