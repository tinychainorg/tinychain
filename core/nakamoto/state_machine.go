package nakamoto

import (
	"database/sql"
)

type CoinStateMachine struct {
	db *sql.DB
}

func NewCoinStateMachine(db *sql.DB) *CoinStateMachine {
	return &CoinStateMachine{
		db: db,
	}
}

func (c *CoinStateMachine) VerifyTxs(tx RawTransaction) error {
	// if tx.index == 0, process as coinbase
	// else process a transfer using the account model
	// update the accounts tree with my new balance.
	return nil
}