package nakamoto

import (
	"testing"

	"github.com/liamzebedee/tinychain-go/core"
	"github.com/stretchr/testify/assert"
)

func TestMempool(t *testing.T) {
	// TODO implement.
}

func isCandidate(t *testing.T, mempool *Mempool, tx *RawTransaction) bool {
	found := false
	for _, i := range mempool.candidateTxns {
		// fmt.Printf("i: %v\n", i)
		if tx == i {
			return true
		}
	}
	return found
}

func TestAddTransaction(t *testing.T) {
	mempool := NewMempool()

	tx1, err1 := newValidTx(t)
	if err1 != nil {
		panic(err1)
	}

	mempool.AddTransaction(&tx1)
	assert.True(t, isCandidate(t, mempool, &tx1))

	tx2, err2 := newValidTx(t)
	if err2 != nil {
		panic(err2)
	}

	mempool.AddTransaction(&tx2)
	assert.True(t, isCandidate(t, mempool, &tx2))
}

func newValidTxWithFee(t *testing.T, fee uint64) (RawTransaction, error) {
	wallets := getTestingWallets(t)

	tx := RawTransaction{
		Version:    1,
		Sig:        [64]byte{},
		FromPubkey: wallets[0].PubkeyBytes(),
		ToPubkey:   [65]byte{},
		Amount:     0,
		Fee:        fee,
		Nonce:      0,
	}

	envelope := tx.Envelope()
	sig, err := wallets[0].Sign(envelope)
	if err != nil {
		return RawTransaction{}, err
	}

	copy(tx.Sig[:], sig)

	// Sanity check verify.
	if !core.VerifySignature(wallets[0].PubkeyStr(), sig, envelope) {
		t.Fatalf("Failed to verify signature.")
	}

	return tx, nil
}

func TestGetFeeRates(t *testing.T) {
	mempool := NewMempool()
	minFee := uint64(10)
	medianFee := uint64(10)
	maxFee := uint64(30)
	tx1, err1 := newValidTxWithFee(t, minFee)
	tx2, err2 := newValidTxWithFee(t, medianFee)
	tx3, err3 := newValidTxWithFee(t, maxFee)
	if err1 != nil || err2 != nil || err3 != nil {
		panic("failed to create txns")
	}
	mempool.AddTransaction(&tx1)
	mempool.AddTransaction(&tx2)
	mempool.AddTransaction(&tx3)
	feeRates := mempool.GetFeeRates()
	assert.Equal(t, minFee, feeRates.MinFee)
	assert.Equal(t, medianFee, feeRates.MedianFee)
	assert.Equal(t, maxFee, feeRates.MaxFee)
}

func TestBuildBundle(t *testing.T) {

}

// note: validators don’t have to trust builders not to withhold block bodies or publish invalid blocks because payment is unconditional. The validator’s fee still processes even if the proposed block is unavailable or declared invalid by other validators. In the latter case, the block is simply discarded, forcing the block builder to lose all transaction fees and MEV revenue.
// https://ethereum.org/en/developers/docs/mev/

// Flow: User -> Node -> Mempool <-> Miner
// Test: from user to node, that the user's submitted transaction makes it to the mempool
// Test: that the builder's transaction verification works
// Test: that the builder's bundle is the right size, and that the transactions are ordered by fee and that the builders commit/reveal is legit
// Test: what happens if there's a reorg? the miner re-broadcasts back to the builders? WHERE'S THE CODE
// Test: how do the fees work?
