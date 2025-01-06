package nakamoto

import (
	"cmp"
	"errors"
	"slices"
)

// The mempool stores transactions that have not yet been confirmed by the network. When a user submits a transaction, it goes into a mempool. Miners request a transaction bundle from the mempool to include in the next block they mine.
//
// Building a bundle of transactions involves a first-price auction for blockspace, whereby transactions are ordered by fee descending and included in the block until the block is full (at maximum capacity).
//
// This design is modelled off of the work done in Ethereum's MEV space, where proposers (miners) receive blocks from builders, who try to maximise their profit through extraction of value (MEV) while also competing on bundle selection by maximising the proposer's profit through fees.
//
// Note that due to how Nakamoto consensus works, there is the possibility of reorgs, which means that a block that was previously mined may be replaced by a longer chain. In this case, transactions which have been taken from the mempool and included in a block that is later reorged out should be "returned" to the mempool. This is the intuition for the mempool's behaviour, however it is designed as a one-way flow.
type Mempool struct {
	txs []*RawTransaction
}

type FeeStatistics struct {
	MinFee    float64
	MedianFee float64
	MaxFee    float64
	MeanFee   float64
}

var ErrFeeTooLow = errors.New("mempool: fee too low")

// The maximum size of the mempool in transactions.
const MempoolMaxSize = 8192

// NewMempool creates a new mempool.
func NewMempool() *Mempool {
	return &Mempool{
		txs: []*RawTransaction{},
	}
}

// Insert transactions into the mempool without validation.
func (m *Mempool) Insert(txs []*RawTransaction) {
	m.txs = append(m.txs, txs...)

	// Sort the mempool by fee descending.
	slices.SortFunc(m.txs, func(i, j *RawTransaction) int {
		return -1 * cmp.Compare(i.Fee, j.Fee)
	})
}

// Add a transaction to the mempool, performing logic checks.
func (m *Mempool) SubmitTx(tx RawTransaction) error {
	if MempoolMaxSize == len(m.txs) {
		// Enact fee policy.
		// Txs are ordered by fee, so the first tx in the mempool has the lowest fee.
		minFee := m.txs[0].Fee
		if tx.Fee <= minFee {
			return ErrFeeTooLow
		}
	}

	m.txs = append(m.txs, &tx)

	// Sort the mempool by fee descending.
	slices.SortFunc(m.txs, func(i, j *RawTransaction) int {
		return -1 * cmp.Compare(i.Fee, j.Fee)
	})

	// Trim the mempool to its max size.
	if MempoolMaxSize < len(m.txs) {
		m.txs = m.txs[0:MempoolMaxSize]
	}

	return nil
}

// Gets the fee statistics for use in fee estimation.
func (m *Mempool) GetFeeStatistics() FeeStatistics {
	stats := FeeStatistics{
		MinFee:    0,
		MedianFee: 0,
		MaxFee:    0,
		MeanFee:   0,
	}

	if len(m.txs) == 0 {
		return stats
	}

	fees := make([]float64, len(m.txs))
	for i, tx := range m.txs {
		fees[i] = float64(tx.Fee)
	}

	// Min.
	stats.MinFee = slices.Min(fees)

	// Median.
	for i, fee := range fees {
		if i >= len(fees)/2 {
			stats.MedianFee = fee
			break
		}
	}

	// Max.
	stats.MaxFee = slices.Max(fees)

	// Mean.
	sum := 0.0
	for _, fee := range fees {
		sum += fee
	}
	stats.MeanFee = sum / float64(len(fees))

	return stats
}

// Creates a bundle from the mempool.
func (m *Mempool) GetBundle(max uint) []RawTransaction {
	if max == 0 {
		return []RawTransaction{}
	}

	bundle := []RawTransaction{}
	for _, tx := range m.txs {
		bundle = append(bundle, *tx)
		if uint(len(bundle)) == max {
			break
		}
	}

	return bundle
}

// Regenerates the mempool from the current state of the chain, removing transactions that are already in the chain.
func (m *Mempool) Regenerate() {
	// TODO.
}
