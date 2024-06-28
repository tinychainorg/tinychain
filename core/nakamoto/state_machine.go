package nakamoto

import (
	"bytes"
	"database/sql"
	"encoding/binary"
)

type TransferOp struct {
	OpName string  `json:"op_name"`
	Amount uint64  `json:"amount"`
	To    [65]byte `json:"to"`
}

type Account struct {
	PubKey [65]byte
	Balance uint64
}

func (t TransferOp) Bytes() []byte {
	buf := new(bytes.Buffer)
	
	err := binary.Write(buf, binary.BigEndian, t.OpName)
	if err != nil { panic(err) }
	err = binary.Write(buf, binary.BigEndian, t.Amount)	
	if err != nil { panic(err) }
	err = binary.Write(buf, binary.BigEndian, t.To)
	if err != nil { panic(err) }

	return buf.Bytes()
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

	// The in-memory state trie.
	Accounts map[[65]byte]Account
}

func NewCoinStateMachine(db *sql.DB) (*CoinStateMachine, error) {
	return &CoinStateMachine{
		db: db,
		Accounts: make(map[[65]byte]Account),
	}, nil
}

func (c *CoinStateMachine) ApplySnapshot(snapshot []Account) {
	for _, account := range snapshot {
		c.Accounts[account.PubKey] = account
	}
}

// Transitions the state machine to the next state.
// State changes are persisted to an in memory pool.
func (c *CoinStateMachine) Transition(input CoinStateMachineInput) error {
	// Decode the data into a To field.
	// Get the pubkey from the To field.
	// Check the `from` account for balance.
	// Deduct the coins from the `from` account balance.
	// Add the coins to the `to` account balance.
	return nil
}

func (c *CoinStateMachine) GetBalance(account [65]byte) (uint64) {
	return 0
}

// Returns a list of modified accounts.
func (c* CoinStateMachine) GetStateSnapshot() ([]Account) {
	return nil
}