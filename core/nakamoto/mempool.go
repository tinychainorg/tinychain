package nakamoto

import (
	"encoding/hex"
	"math"
	"sort"

	"github.com/liamzebedee/tinychain-go/core"
)

// The mempool stores transactions that have not yet been confirmed by the network. When a user submits a transaction, it goes into a mempool. Miners request a transaction bundle from the mempool to include in the next block they mine.
//
// Building a bundle of transactions involves an auction for blockspace, whereby
// transactions are ordered by fee and included in the block until the block is full (at maximum capacity).
//
// This design is modelled off of the work done in Ethereum's MEV space, where proposers (miners) receive blocks from builders, who try to maximise their profit through extraction of value (MEV) while also competing on bundle selection by maximising the proposer's profit through fees.
//
// Note that due to how Nakamoto consensus works, there is the possibility of reorgs, which means that a block that was previously mined may be replaced by a longer chain. In this case, transactions which have been taken from the mempool and included in a block that is later reorged out should be "returned" to the mempool. This is the intuition for the mempool's behaviour, however it is designed as a one-way flow.
type Mempool struct {
	candidateTxns []*RawTransaction
	confirmedTxns []*Transaction
	fees          *FeeRates
}

type FeeRates struct {
	MinFee    uint64
	MedianFee uint64
	MaxFee    uint64
}

// NewMempool creates a new mempool.
func NewMempool() *Mempool {
	return &Mempool{
		candidateTxns: []*RawTransaction{},
		confirmedTxns: []*Transaction{},
		fees: &FeeRates{
			MinFee:    math.MaxUint64,
			MedianFee: 0,
			MaxFee:    0,
		},
	}
}

func (m *Mempool) insertCandidate(tx *RawTransaction) {
	index := sort.Search(len(m.candidateTxns), func(i int) bool {
		return m.candidateTxns[i].Fee >= tx.Fee
	})

	// fmt.Printf("index: %d\n", index)

	newArr := make([]*RawTransaction, len(m.candidateTxns)+1)
	copy(newArr[:index], m.candidateTxns[:index])
	newArr[index] = tx
	copy(newArr[index+1:], m.candidateTxns[index:])
	m.candidateTxns = newArr
}

func (m *Mempool) setMedianFee() {
	length := len(m.candidateTxns)
	if length == 0 {
		return
	}
	if length%2 == 0 {
		m.fees.MedianFee = (m.candidateTxns[length/2-1].Fee + m.candidateTxns[length/2].Fee) / 2
	} else if length%2 == 1 {
		m.fees.MedianFee = m.candidateTxns[length/2].Fee
	}
}

func isValidTx(tx *RawTransaction) bool {
	validSig := core.VerifySignature(hex.EncodeToString(tx.FromPubkey[:]), tx.Sig[:], tx.Envelope())
	return validSig
}

func (m *Mempool) AddTransaction(tx *RawTransaction) {
	if !isValidTx(tx) {
		return // skip for now
	}

	m.insertCandidate(tx)

	m.fees.MinFee = m.candidateTxns[0].Fee
	m.setMedianFee()
	m.fees.MaxFee = m.candidateTxns[len(m.candidateTxns)-1].Fee
}

func (m *Mempool) GetFeeRates() FeeRates {
	return *m.fees
}

func (m *Mempool) BuildBundle() []*RawTransaction {
	txs := []*RawTransaction{}
	return txs
}

// event: n.Dag.OnNewFullTip
