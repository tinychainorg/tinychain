// 
// The mempool stores transactions that have not yet been confirmed by the network. When a user submits a transaction, it goes into a mempool. Miners request a transaction bundle from the mempool to include in the next block they mine.
// 
// Building a bundle of transactions involves an auction for blockspace, whereby
// transactions are ordered by fee and included in the block until the block is full (at maximum capacity).
// 
// This design is modelled off of the work done in Ethereum's MEV space, where proposers (miners) receive blocks from builders, who try to maximise their profit through extraction of value (MEV) while also competing on bundle selection by maximising the proposer's profit through fees.
// 
// Note that due to how Nakamoto consensus works, there is the possibility of reorgs, which means that a block that was previously mined may be replaced by a longer chain. In this case, transactions which have been taken from the mempool and included in a block that is later reorged out should be "returned" to the mempool. This is the intuition for the mempool's behaviour, however it is designed as a one-way flow.
// 
package core

type Mempool struct {
}

type FeeInfo struct {
	MinFee uint64
	MedianFee uint64
	MaxFee uint64
}

// NewMempool creates a new mempool.
func NewMempool() *Mempool {
	return &Mempool{}
}

func (m *Mempool) AddTransaction(tx *Transaction) {}

func (m *Mempool) GetFeeRates() FeeInfo {}

func (m *Mempool) BuildBundle() []*Transaction {
	txs := []*Transaction{}
	return txs
}