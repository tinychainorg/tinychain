package nakamoto

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

func OpenDB(dbPath string) (*sql.DB, error) {
	logger := NewLogger("blockdag", "db")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	tx, err := db.Begin()

	// Check to perform migrations.
	_, err = tx.Exec("create table if not exists tinychain_version (version int)")
	if err != nil {
		return nil, fmt.Errorf("error checking database version: %s", err)
	}
	// Check the database version.
	rows, err := tx.Query("select version from tinychain_version order by version asc limit 1")
	if err != nil {
		return nil, fmt.Errorf("error checking database version: %s", err)
	}
	databaseVersion := 0
	if rows.Next() {
		rows.Scan(&databaseVersion)
	}
	err = rows.Close()
	if err != nil {
		return nil, err
	}

	// Log version.
	logger.Printf("Database version: %d\n", databaseVersion)

	// Migration: v0.
	if databaseVersion == 0 {
		// Perform migrations.
		dbVersion := 1
		logger.Printf("Running migration: %d\n", dbVersion)

		// Create tables.

		// epochs
		_, err = tx.Exec(`create table epochs (
			id TEXT PRIMARY KEY, 
			start_block_hash blob, 
			start_time integer, 
			start_height integer, 
			difficulty blob
		)`)
		if err != nil {
			return nil, fmt.Errorf("error creating 'epochs' table: %s", err)
		}

		// blocks
		_, err = tx.Exec(`create table blocks (
			hash blob primary key, 
			parent_hash blob, 
			difficulty blob, 
			timestamp integer, 
			num_transactions integer, 
			transactions_merkle_root blob, 
			nonce blob, 
			graffiti blob, 
			height integer, 
			epoch TEXT, 
			size_bytes integer, 
			parent_total_work blob, 
			acc_work blob, 
			foreign key (epoch) REFERENCES epochs (id)
		)`)
		if err != nil {
			return nil, fmt.Errorf("error creating 'blocks' table: %s", err)
		}

		// transactions_blocks
		_, err = tx.Exec(`
			create table transactions_blocks (
				block_hash blob, transaction_hash blob, txindex integer, 
				
				primary key (block_hash, transaction_hash, txindex),
				foreign key (block_hash) references blocks (hash), 
				foreign key (transaction_hash) references transactions (hash)
			)
		`)
		if err != nil {
			return nil, fmt.Errorf("error creating 'transactions_blocks' table: %s", err)
		}

		// transactions
		_, err = tx.Exec(`create table transactions (
			hash blob primary key, 
			sig blob, 
			from_pubkey blob, 
			to_pubkey blob, 
			amount integer, 
			fee integer, 
			nonce integer, 
			version integer
		)`)
		if err != nil {
			return nil, fmt.Errorf("error creating 'transactions' table: %s", err)
		}

		// Create indexes.
		_, err = tx.Exec(`create index blocks_parent_hash on blocks (parent_hash)`)
		if err != nil {
			return nil, fmt.Errorf("error creating 'blocks_parent_hash' index: %s", err)
		}

		// Update version.
		_, err = tx.Exec("insert into tinychain_version (version) values (?)", dbVersion)
		if err != nil {
			return nil, fmt.Errorf("error updating database version: %s", err)
		}

		logger.Printf("Database upgraded to: %d\n", dbVersion)
	}
	if databaseVersion < 1 {
		_, err = tx.Exec(
			`CREATE INDEX idx_blocks_parent_hash ON blocks (parent_hash);
			CREATE INDEX idx_blocks_hash ON blocks (hash);
			CREATE INDEX idx_blocks_acc_work ON blocks (acc_work);
			CREATE INDEX idx_transactions_blocks_block_hash ON transactions_blocks (block_hash);
			CREATE INDEX idx_transactions_blocks_transaction_hash ON transactions_blocks (transaction_hash);
			CREATE INDEX idx_transactions_blocks_txindex ON transactions_blocks (txindex);
		`)
		if err != nil {
			return nil, fmt.Errorf("error creating indexes: %s", err)
		}

		// Update version.
		dbVersion := 2
		_, err = tx.Exec("insert into tinychain_version (version) values (?)", dbVersion)
		if err != nil {
			return nil, fmt.Errorf("error updating database version: %s", err)
		}
		logger.Printf("Database upgraded to: %d\n", dbVersion)
	}
	if databaseVersion < 3 {
		// config
		_, err = tx.Exec(`create table datastores (
			id TEXT PRIMARY KEY, 
			data blob
		)`)
		if err != nil {
			return nil, fmt.Errorf("error creating 'epochs' table: %s", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		panic(err)
	}

	return db, err
}

type NetworkStore struct {
	// A cache of peers we have connected to.
	Peers map[string]*Peer
}

type WalletsStore struct {
	Wallets []UserWallet `json:"wallets"`
}

type UserWallet struct {
	// Wallet label.
	Label string `json:"label"`
	// Wallet private key.
	PrivateKeyString string `json:"privateKeyString"`
}

func LoadConfigStore[T NetworkStore | WalletsStore](db *sql.DB, key string) (*T, error) {
	buf := []byte("{}")
	err := db.QueryRow("SELECT data FROM datastores WHERE id = ?", key).Scan(&buf)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Decode the data into the store.
	var store T
	err = json.Unmarshal(buf, &store)
	if err != nil {
		return nil, err
	}
	
	return &store, nil
}

func SaveConfigStore[T NetworkStore | WalletsStore](db *sql.DB, key string, value T) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	
	// Encode the store into a byte slice.
	buf, err := json.Marshal(value)
	if err != nil {
		return err
	}

	// Perform an upsert (insert or update) operation.
	_, err = tx.Exec("INSERT INTO datastores (id, data) VALUES (?, ?) ON CONFLICT(id) DO UPDATE SET data = excluded.data", key, buf)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}