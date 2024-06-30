package nakamoto

import (
	"database/sql"
	"errors"
	"math/bits"
)

var ErrInsufficientBalance = errors.New("insufficient balance")
var ErrToBalanceOverflow = errors.New("\"to\" balance overflow")

type StateLeaf struct {
	PubKey  [65]byte
	Balance uint64
}

// The input to the state transition function.
type CoinStateMachineInput struct {
	// The raw transaction to be processed.
	RawTransaction RawTransaction

	// Is it the coinbase transaction.
	IsCoinbase bool
}

type CoinStateMachine struct {
	db *sql.DB

	// The current state.
	state map[[65]byte]uint64
}

func NewCoinStateMachine(db *sql.DB) (*CoinStateMachine, error) {
	return &CoinStateMachine{
		db: db,
		state: make(map[[65]byte]uint64),
	}, nil
}

func (c *CoinStateMachine) Apply(leafs []*StateLeaf) {
	for _, leaf := range leafs {
		c.state[leaf.PubKey] = leaf.Balance
	}
}

// Transitions the state machine to the next state.
func (c *CoinStateMachine) Transition(input CoinStateMachineInput) ([]*StateLeaf, error) {
	// Check transaction version.
	if input.RawTransaction.Version != 1 {
		return nil, errors.New("unsupported transaction version")
	}
	
	if input.IsCoinbase {
		return c.transitionCoinbase(input)
	}

	fromBalance := c.GetBalance(input.RawTransaction.FromPubkey)
	toBalance := c.GetBalance(input.RawTransaction.ToPubkey)
	amount := input.RawTransaction.Amount

	// NO OP transfer.
	if amount == 0 {
		return []*StateLeaf{}, nil
	}

	// Check if the `from` account has enough balance.
	if fromBalance < amount {
		return nil, ErrInsufficientBalance
		// return nil, fmt.Errorf("insufficient balance. balance=%d, amount=%d", fromBalance, amount)
	}

	// Check if the `to` balance will overflow.
	// The Add64 function adds two 64-bit unsigned integers along with an optional carry-in value. It returns the result of the addition and the carry-out value. The carry-out is set to 1 if the addition results in an overflow (i.e., the sum is greater than what can be represented in 64 bits), and 0 otherwise.
	if _, carry := bits.Add64(toBalance, amount, 0); carry != 0 {
		return nil, ErrToBalanceOverflow
	}

	// Deduct the coins from the `from` account balance.
	fromBalance -= amount

	// Add the coins to the `to` account balance.
	toBalance += amount
	
	// Create the new state leaves.
	fromLeaf := &StateLeaf{
		PubKey:  input.RawTransaction.FromPubkey,
		Balance: fromBalance,
	}
	toLeaf := &StateLeaf{
		PubKey:  input.RawTransaction.ToPubkey,
		Balance: toBalance,
	}
	leaves := []*StateLeaf{
		fromLeaf,
		toLeaf,
	}
	return leaves, nil
}

func (c *CoinStateMachine) transitionCoinbase(input CoinStateMachineInput) ([]*StateLeaf, error) {
	toBalance := c.GetBalance(input.RawTransaction.ToPubkey)
	amount := input.RawTransaction.Amount

	// NO OP transfer.
	if amount == 0 {
		return []*StateLeaf{}, nil
	}

	// Check if the `to` balance will overflow.
	// The Add64 function adds two 64-bit unsigned integers along with an optional carry-in value. It returns the result of the addition and the carry-out value. The carry-out is set to 1 if the addition results in an overflow (i.e., the sum is greater than what can be represented in 64 bits), and 0 otherwise.
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


func (c *CoinStateMachine) GetBalance(account [65]byte) uint64 {
	return c.state[account]
}

// Returns a list of modified accounts.
func (c *CoinStateMachine) GetStateSnapshot() []StateLeaf {
	return nil
}
