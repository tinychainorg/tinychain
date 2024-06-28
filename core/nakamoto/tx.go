package nakamoto

import "github.com/liamzebedee/tinychain-go/core"

func MakeTransferTx(from [65]byte, to [65]byte, amount uint64, wallet *core.Wallet) RawTransaction {
	op := TransferOp{
		OpName: "transfer",
		Amount: amount,
		To:     to,
	}
	tx := RawTransaction{
		FromPubkey: from,
		Sig:        [64]byte{},
		Data:       op.Bytes(),
	}
	// Sign tx.
	sig, err := wallet.Sign(tx.Data)
	if err != nil {
		panic(err)
	}
	copy(tx.Sig[:], sig)
	return tx
}
