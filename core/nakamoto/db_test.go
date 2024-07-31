package nakamoto

import (
	"testing"

	"github.com/liamzebedee/tinychain-go/core"
)

func TestOpenDB(t *testing.T) {
	t.Log("Testing database open")

	// Open an in-memory database.
	db, err := OpenDB(":memory:")
	if err != nil {
		t.Error(err)
		return
	}
	defer db.Close()

	// Check that the database is open.
	err = db.Ping()
	if err != nil {
		t.Error(err)
	}
}

func TestLoadSaveConfigStore(t *testing.T) {
	t.Log("Testing config store")

	db, err := OpenDB(":memory:")
	if err != nil {
		t.Error(err)
		return
	}
	defer db.Close()

	// Create a new config store.
	wallets, err := LoadDataStore[WalletsStore](db, "wallets")
	if err != nil {
		t.Error(err)
		return
	}

	// Check that the store is empty.
	if len(wallets.Wallets) != 0 {
		t.Error("wallets store should be empty")
		return
	}

	// Add a wallet to the store.
	wallet, err := core.CreateRandomWallet()
	if err != nil {
		t.Error(err)
		return
	}
	wallets.Wallets = append(wallets.Wallets, UserWallet{
		Label: "testwallet",
		PrivateKeyString: wallet.PrvkeyStr(),
	})

	// Save the store.
	err = SaveDataStore[WalletsStore](db, "wallets", *wallets)
	if err != nil {
		t.Error(err)
		return
	}

	// Load the store again.
	wallets, err = LoadDataStore[WalletsStore](db, "wallets")
	if err != nil {
		t.Error(err)
		return
	}

	// Check that the store has one wallet.
	if len(wallets.Wallets) != 1 {
		t.Error("wallets store should have one wallet")
		return
	}

	// Check that the wallet is correct.
	if wallets.Wallets[0].Label != "testwallet" {
		t.Error("wallet label is incorrect")
		return
	}

	// Check that the wallet private key is correct.
	if wallets.Wallets[0].PrivateKeyString != wallet.PrvkeyStr() {
		t.Error("wallet private key is incorrect")
		return
	}

	// Test saving a store overwrites an old store.
	err = SaveDataStore[WalletsStore](db, "wallets", *wallets)
	if err != nil {
		t.Error(err)
		return
	}
}