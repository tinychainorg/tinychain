package nakamoto

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/fatih/color"
)

func dbGetVersion(db *sql.DB) (int, error) {
	// Check the database version.
	row := db.QueryRow("SELECT version FROM tinychain_version ORDER BY version DESC LIMIT 1")
	if err := row.Err(); err != nil {
		return -1, fmt.Errorf("error checking database version: %s", err)
	}
	
	databaseVersion := -1
	row.Scan(&databaseVersion)

	return databaseVersion, nil
}

func dbMigrate(db *sql.DB, migrationIndex int, migrateFn func(tx *sql.Tx) error) error {
	logger := NewLogger("db", "")

	version, err := dbGetVersion(db)
	if err != nil {
		return err
	}

	// Skip migration if the database is already at the target version.
	if migrationIndex <= version { 
		logger.Printf("Skipping migration: %d\n", migrationIndex)
		return nil 
	}

	// Perform the migration.
	logger.Printf("Running migration: %d\n", migrationIndex)
	tx, err := db.Begin()
	err = migrateFn(tx)
	if err != nil {
		tx.Rollback()
		return err
	}

	// Update the database version.
	_, err = tx.Exec("insert into tinychain_version (version) values (?)", migrationIndex)
	if err != nil {
		tx.Rollback()
		return err
	}

	err = tx.Commit()
	if err != nil {
		panic(err)
	}

	return nil
}

func OpenDB(dbPath string) (*sql.DB, error) {
	logger := NewLogger("db", "")

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// Check to perform migrations.
	_, err = db.Exec("create table if not exists tinychain_version (version int)")
	if err != nil {
		return nil, fmt.Errorf("error checking database version: %s", err)
	}
	// Check the database version.
	databaseVersion, err := dbGetVersion(db)
	if err != nil {
		return nil, fmt.Errorf("error checking database version: %s", err)
	}

	// Log version.
	logger.Printf("Database version: %d\n", databaseVersion)

	// Migration: v0.
	dbMigrate(db, 0, func(tx *sql.Tx) error {
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
			return fmt.Errorf("error creating 'epochs' table: %s", err)
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
			return fmt.Errorf("error creating 'blocks' table: %s", err)
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
			return fmt.Errorf("error creating 'transactions_blocks' table: %s", err)
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
			return fmt.Errorf("error creating 'transactions' table: %s", err)
		}

		// Create indexes.
		_, err = tx.Exec(`create index blocks_parent_hash on blocks (parent_hash)`)
		if err != nil {
			return fmt.Errorf("error creating 'blocks_parent_hash' index: %s", err)
		}

		return nil
	})

	dbMigrate(db, 1, func(tx *sql.Tx) error {
		_, err = tx.Exec(
			`CREATE INDEX idx_blocks_parent_hash ON blocks (parent_hash);
			CREATE INDEX idx_blocks_hash ON blocks (hash);
			CREATE INDEX idx_blocks_acc_work ON blocks (acc_work);
			CREATE INDEX idx_transactions_blocks_block_hash ON transactions_blocks (block_hash);
			CREATE INDEX idx_transactions_blocks_transaction_hash ON transactions_blocks (transaction_hash);
			CREATE INDEX idx_transactions_blocks_txindex ON transactions_blocks (txindex);
		`)
		if err != nil {
			return fmt.Errorf("error creating indexes: %s", err)
		}
		return nil
	})

	dbMigrate(db, 2, func(tx *sql.Tx) error {
		// config
		_, err = tx.Exec(`create table datastores (
			-- use k,v instead of key,value to avoid reserved word conflicts
			k TEXT PRIMARY KEY, 
			v blob
		)`)
		if err != nil {
			return fmt.Errorf("error creating 'datastores' table: %s", err)
		}
		return nil
	})

	return db, err
}

// DataStore is a generic interface for reading/writing persistent data to the database.
// It is used for storing configuration (wallet private keys), caching (peer addresses) and other things. They are stored in the database under a unique key, and are serialised/deserialised using the JSON encoding.
type DataStore interface {
	NetworkStore | WalletsStore
}

type NetworkStore struct {
	// A cache of peers we have connected to.
	PeerCache []Peer `json:"peerCache"`
}

type WalletsStore struct {
	Wallets []UserWallet `json:"wallets"`
}

type UserWallet struct {
	// Wallet label.
	Label string `json:"label"`
	// Wallet private key as a hex string.
	PrivateKeyString string `json:"privateKeyString"`
}

// Load a data store from the database by key.
func LoadDataStore[T DataStore](db *sql.DB, key string) (*T, error) {
	logger := NewLogger("db", "")

	buf := []byte("{}")
	err := db.QueryRow("SELECT v FROM datastores WHERE k = ?", key).Scan(&buf)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Decode the data into the store.
	var store T
	err = json.Unmarshal(buf, &store)
	if err != nil {
		return nil, err
	}

	logger.Printf("store name=%s loaded\n", color.HiYellowString(key))
	
	return &store, nil
}

// Persist a data store to the database under the given key.
func SaveDataStore[T DataStore](db *sql.DB, key string, value T) error {
	logger := NewLogger("db", "")
	logger.Printf("store name=%s saving\n", color.HiYellowString(key))

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
	_, err = tx.Exec("INSERT INTO datastores (k, v) VALUES (?, ?) ON CONFLICT(k) DO UPDATE SET v = excluded.v", key, buf)
	if err != nil {
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	logger.Printf("store name=%s saved\n", color.HiYellowString(key))

	return nil
}