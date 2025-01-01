package nakamoto

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMempoolSubmitTxSorted(t *testing.T) {
	mempool := NewMempool()
	// Insert 3 txs of varying fees.
	tx1 := newValidTxWithFee(t, 100, 1)
	tx2 := newValidTxWithFee(t, 100, 2)
	tx3 := newValidTxWithFee(t, 100, 3)
	mempool.Insert([]*RawTransaction{&tx2, &tx1, &tx3})
	// Check sort order (fee descending).
	assert.Equal(t, &tx3, mempool.txs[0])
	assert.Equal(t, &tx2, mempool.txs[1])
	assert.Equal(t, &tx1, mempool.txs[2])

	// Submit 1 tx.
	tx4 := newValidTxWithFee(t, 100, 2)
	err := mempool.SubmitTx(tx4)
	assert.NoError(t, err)

	// Check sort order (fee descending).
	assert.Equal(t, &tx3, mempool.txs[0])
	assert.Equal(t, &tx2, mempool.txs[1])
	assert.Equal(t, &tx4, mempool.txs[2])
	assert.Equal(t, &tx1, mempool.txs[3])

	// Submit 1 tx.
	tx5 := newValidTxWithFee(t, 100, 30)
	err = mempool.SubmitTx(tx5)
	assert.NoError(t, err)
	assert.Equal(t, &tx5, mempool.txs[0])
	assert.Equal(t, &tx3, mempool.txs[1])
	assert.Equal(t, &tx2, mempool.txs[2])
	assert.Equal(t, &tx4, mempool.txs[3])
	assert.Equal(t, &tx1, mempool.txs[4])
}

func TestMempoolSubmitTx(t *testing.T) {
	// TC1 - Empty mempool.
	mempool1 := NewMempool()
	tx1 := newValidTxWithFee(t, 100, 0)
	err := mempool1.SubmitTx(tx1)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(mempool1.txs))
	assert.Equal(t, &tx1, mempool1.txs[0])

	// TC2 - Mempool with 1 tx.
	mempool2 := mempool1
	tx2 := newValidTxWithFee(t, 100, 0)
	err = mempool2.SubmitTx(tx2)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(mempool2.txs))
	assert.Equal(t, &tx2, mempool2.txs[1])

	// TC3-4 - Max mempool size, fee rate kicks in.

	// TC3 - tx with fee 0 fails.
	mempool3 := NewMempool()
	for i := 0; i < MempoolMaxSize; i++ {
		tx := newValidTxWithFee(t, 100, 0)
		err = mempool3.SubmitTx(tx)
		assert.NoError(t, err)
	}
	tx3 := newValidTxWithFee(t, 100, 0)
	err = mempool3.SubmitTx(tx3)
	assert.Equal(t, ErrFeeTooLow.Error(), err.Error())

	// TC4 - tx with fee 1 succeeds.
	tx4 := newValidTxWithFee(t, 100, 1)
	err = mempool3.SubmitTx(tx4)
	assert.NoError(t, err)
	assert.Equal(t, 8192, len(mempool3.txs))
	assert.Equal(t, &tx4, mempool3.txs[0])
}

func TestMempoolEmptyGetFeeStatistics(t *testing.T) {
	mempool1 := NewMempool()
	stats := mempool1.GetFeeStatistics()
	assert.Equal(t, 0.0, stats.MinFee)
	assert.Equal(t, 0.0, stats.MedianFee)
	assert.Equal(t, 0.0, stats.MaxFee)
	assert.Equal(t, 0.0, stats.MeanFee)
}

func TestMempoolGetFeeStatistics(t *testing.T) {
	mempool1 := NewMempool()
	// Insert 3 txs of varying fees.
	tx1 := newValidTxWithFee(t, 100, 1)
	tx2 := newValidTxWithFee(t, 100, 2)
	tx3 := newValidTxWithFee(t, 100, 3)
	mempool1.SubmitTx(tx1)
	mempool1.SubmitTx(tx2)
	mempool1.SubmitTx(tx3)

	stats := mempool1.GetFeeStatistics()
	assert.Equal(t, 1.0, stats.MinFee)
	assert.Equal(t, 2.0, stats.MedianFee)
	assert.Equal(t, 3.0, stats.MaxFee)
	assert.Equal(t, 2.0, stats.MeanFee)
}

func newValidTxWithFee(t *testing.T, amt, fee uint64) RawTransaction {
	wallets := getTestingWallets(t)

	tx := RawTransaction{
		Version:    1,
		Sig:        [64]byte{},
		FromPubkey: wallets[0].PubkeyBytes(),
		ToPubkey:   [65]byte{},
		Amount:     amt,
		Fee:        fee,
		Nonce:      randomNonce(),
	}

	envelope := tx.Envelope()
	sig, err := wallets[0].Sign(envelope)
	if err != nil {
		panic(err)
	}

	copy(tx.Sig[:], sig)

	return tx
}
