package nakamoto

import (
	"database/sql"
	"errors"
	"fmt"
	"math/bits"
)

var ErrInsufficientBalance = errors.New("insufficient balance")
var ErrToBalanceOverflow = errors.New("\"to\" balance overflow")
var ErrMinerBalanceOverflow = errors.New("\"miner\" balance overflow")
var ErrAmountPlusFeeOverflow = errors.New("(amount + fee) overflow")
var ErrTxAlreadySequenced = errors.New("transaction already sequenced")

var stateMachineLogger = NewLogger("state-machine", "")

type StateLeaf struct {
	PubKey  [65]byte
	Balance uint64
}

// The input to the state transition function.
type StateMachineInput struct {
	// The raw transaction to be processed.
	RawTransaction RawTransaction

	// Is it the coinbase transaction.
	IsCoinbase bool

	// Miner address for fees.
	MinerPubkey [65]byte
}

// The state machine is the core of the business logic for the Nakamoto blockchain.
// It performs the state transition function, which encapsulates:
// 1. Minting coins into circulation via the coinbase transaction.
// 2. Transferring coins between accounts.
//
// It is oblivious to:
//   - the consensus algorithm, transaction sequencing.
//   - signatures. The state machine does not care about validating signatures. At Bitcoin's core, it is a sequencing/DA layer.
type StateMachine struct {
	// The current state.
	state map[[65]byte]uint64
}

func NewStateMachine(db *sql.DB) (*StateMachine, error) {
	return &StateMachine{
		state: make(map[[65]byte]uint64),
	}, nil
}

func (c *StateMachine) Apply(leafs []*StateLeaf) {
	for _, leaf := range leafs {
		c.state[leaf.PubKey] = leaf.Balance
	}
}

// Transitions the state machine to the next state.
func (c *StateMachine) Transition(input StateMachineInput) ([]*StateLeaf, error) {
	// Check transaction version.
	if input.RawTransaction.Version != 1 {
		return nil, errors.New("unsupported transaction version")
	}

	if input.IsCoinbase {
		return c.transitionCoinbase(input)
	} else {
		return c.transitionTransfer(input)
	}
}

func (c *StateMachine) transitionTransfer(input StateMachineInput) ([]*StateLeaf, error) {
	fromAcc := input.RawTransaction.FromPubkey
	toAcc := input.RawTransaction.ToPubkey
	minerAcc := input.MinerPubkey

	fromBalance := c.GetBalance(fromAcc)
	toBalance := c.GetBalance(toAcc)
	minerBalance := c.GetBalance(minerAcc)

	amount := input.RawTransaction.Amount
	fee := input.RawTransaction.Fee

	tmpState := make(map[[65]byte]uint64)
	tmpState[fromAcc] = fromBalance
	tmpState[toAcc] = toBalance
	tmpState[minerAcc] = minerBalance

	// (1) from account.
	// Check if the amount + fee will overflow.
	if _, carry := bits.Add64(amount, fee, 0); carry != 0 {
		return nil, ErrAmountPlusFeeOverflow
	}
	// Check if the `from` account has enough balance.
	if tmpState[fromAcc] < (amount + fee) {
		return nil, ErrInsufficientBalance
	}
	// Deduct the coins from the `from` account balance.
	tmpState[fromAcc] -= amount
	tmpState[fromAcc] -= fee

	// (2) to account.
	// Check if the `to` account will overflow.
	if _, carry := bits.Add64(tmpState[toAcc], amount, 0); carry != 0 {
		return nil, ErrToBalanceOverflow
	}
	// Add the coins to the `to` account balance.
	tmpState[toAcc] += amount

	// (3) miner account.
	// Check if the `miner` account will overflow.
	if _, carry := bits.Add64(tmpState[minerAcc], fee, 0); carry != 0 {
		return nil, ErrMinerBalanceOverflow
	}
	// Add the fee to the miner account balance.
	tmpState[minerAcc] += fee

	stateMachineLogger.Printf("Transitioning transfer: from=%x to=%x miner=%x amount=%d fee=%d", input.RawTransaction.FromPubkey, input.RawTransaction.ToPubkey, input.MinerPubkey, amount, fee)

	stateMachineLogger.Printf("New balances: from=%d to=%d miner=%d", tmpState[fromAcc], tmpState[toAcc], tmpState[minerAcc])

	// Create the new state leaves.
	leaves := []*StateLeaf{}
	for acc, balance := range tmpState {
		leaves = append(leaves, &StateLeaf{
			PubKey:  acc,
			Balance: balance,
		})
	}
	stateMachineLogger.Printf("New leaves: %d leaves", len(leaves))

	return leaves, nil
}

func (c *StateMachine) transitionCoinbase(input StateMachineInput) ([]*StateLeaf, error) {
	toBalance := c.GetBalance(input.RawTransaction.ToPubkey)
	amount := input.RawTransaction.Amount

	// Check if the `to` balance will overflow.
	if _, carry := bits.Add64(toBalance, amount, 0); carry != 0 {
		return nil, ErrToBalanceOverflow
	}

	// Add the coins to the `to` account balance.
	toBalance += amount

	// Create the new state leaves.
	toLeaf := &StateLeaf{
		PubKey:  input.RawTransaction.ToPubkey,
		Balance: toBalance,
	}
	leaves := []*StateLeaf{
		toLeaf,
	}
	return leaves, nil
}

func (c *StateMachine) GetBalance(account [65]byte) uint64 {
	return c.state[account]
}

// Returns a list of modified accounts.
func (c *StateMachine) GetStateSnapshot() []StateLeaf {
	return nil
}

func (c *StateMachine) GetState() map[[65]byte]uint64 {
	return c.state
}

// Given a block DAG and a list of block hashes, extracts the transaction sequence, applies each transaction in order, and returns the final state.
func RebuildState(dag *BlockDAG, stateMachine StateMachine, longestChainHashList [][32]byte) (*StateMachine, error) {
	for _, blockHash := range longestChainHashList {
		// 1. Get all transactions for block.
		// TODO ignore: nonce, sig
		txs, err := dag.GetBlockTransactions(blockHash)
		if err != nil {
			return nil, err
		}

		// stateMachineLogger.Printf("Processing block %x with %d transactions", blockHash, len(*txs))

		// 2. Map transactions to state leaves through state machine transition function.
		var stateMachineInput StateMachineInput
		var minerPubkey [65]byte
		isCoinbase := false

		for i, tx := range *txs {
			// Special case: coinbase tx is always the first tx in the block.
			if i == 0 {
				minerPubkey = tx.FromPubkey
				isCoinbase = true
			}

			// Construct the state machine input.
			stateMachineInput = StateMachineInput{
				RawTransaction: tx.ToRawTransaction(),
				IsCoinbase:     isCoinbase,
				MinerPubkey:    minerPubkey,
			}

			// Transition the state machine.
			effects, err := stateMachine.Transition(stateMachineInput)
			if err != nil {
				return nil, fmt.Errorf("Error transitioning state machine: block=%x txindex=%d error=\"%s\"", blockHash, i, err)
			}

			// Apply the effects.
			stateMachine.Apply(effects)

			if i == 0 {
				isCoinbase = false
			}
		}
	}

	return &stateMachine, nil
}
