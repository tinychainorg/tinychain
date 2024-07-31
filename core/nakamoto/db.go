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
	_, err = tx.Exec("INSERT INTO tinychain_version (version) VALUES (?)", migrationIndex)
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
	_, err = db.Exec("CREATE TABLE IF NOT EXISTS tinychain_version (version INT)")
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
		_, err = tx.Exec(`CREATE TABLE epochs (
			id TEXT PRIMARY KEY, 
			start_block_hash BLOB, 
			start_time INTEGER, 
			start_height INTEGER, 
			difficulty BLOB
		)`)
		if err != nil {
			return fmt.Errorf("error creating 'epochs' table: %s", err)
		}

		// blocks
		_, err = tx.Exec(`CREATE TABLE blocks (
			hash BLOB PRIMARY KEY, 
			parent_hash BLOB, 
			difficulty BLOB, 
			timestamp INTEGER, 
			num_transactions BLOB, 
			transactions_merkle_root BLOB, 
			nonce BLOB, 
			graffiti BLOB, 
			height INTEGER, 
			epoch TEXT, 
			size_bytes INTEGER, 
			parent_total_work BLOB, 
			acc_work BLOB, 
			FOREIGN KEY (epoch) REFERENCES epochs (id)
		)`)
		if err != nil {
			return fmt.Errorf("error creating 'blocks' table: %s", err)
		}

		// transactions_blocks
		_, err = tx.Exec(`
			CREATE TABLE transactions_blocks (
				block_hash BLOB, transaction_hash BLOB, txindex INTEGER, 
				
				PRIMARY KEY (block_hash, transaction_hash, txindex),
				FOREIGN KEY (block_hash) REFERENCES blocks (hash), 
				FOREIGN KEY (transaction_hash) REFERENCES transactions (hash)
			)
		`)
		if err != nil {
			return fmt.Errorf("error creating 'transactions_blocks' table: %s", err)
		}

		// transactions
		_, err = tx.Exec(`CREATE TABLE transactions (
			hash BLOB PRIMARY KEY, 
			sig BLOB, 
			from_pubkey BLOB, 
			to_pubkey BLOB, 
			amount INTEGER, 
			fee INTEGER, 
			nonce INTEGER, 
			version INTEGER
		)`)
		if err != nil {
			return fmt.Errorf("error creating 'transactions' table: %s", err)
		}

		// Create indexes.
		_, err = tx.Exec(`CREATE INDEX blocks_parent_hash ON blocks (parent_hash)`)
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
		_, err = tx.Exec(`CREATE TABLE datastores (
			-- use k,v instead of key,value to avoid reserved word conflicts
			k TEXT PRIMARY KEY, 
			v BLOB
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
