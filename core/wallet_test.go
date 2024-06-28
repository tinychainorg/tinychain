package core

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

func TestCreateWallet(t *testing.T) {
	// test not null
	wallet, err := CreateRandomWallet()
	if err != nil {
		t.Fatalf("Failed to create wallet: %s", err)
	}
	if wallet == nil {
		t.Fatalf("Failed to create wallet")
	}

	// Print wallet details.
	t.Logf("Wallet:")
	t.Logf("  Pubkey: %s", wallet.PubkeyStr())
	t.Logf("  Prvkey: %s", wallet.PrvkeyStr())
	t.Logf("  Address: %s", wallet.Address())
}

func TestSign(t *testing.T) {
	assert := assert.New(t)

	wallet, err := WalletFromPrivateKey("2053e3c0d239d12a554ef55895b89e5d044af7d09d8be9a8f6da22460f8260ca")
	if err != nil {
		t.Fatalf("Failed to create wallet: %s", err)
	}

	// Sign a message.
	msg := []byte("Gday, world!")
	sig, err := wallet.Sign(msg)
	if err != nil {
		t.Fatalf("Failed to sign message: %s", err)
	}

	t.Logf("Signature: %s", hex.EncodeToString(sig))

	// Verify the signature.
	pubkey := wallet.Pubkey()
	hash := sha256.Sum256(msg)
	r := new(big.Int).SetBytes(sig[:len(sig)/2])
	s := new(big.Int).SetBytes(sig[len(sig)/2:])
	ok := ecdsa.Verify(pubkey, hash[:], r, s)

	assert.True(ok)
}

func TestVerifyWithRealSig(t *testing.T) {
	assert := assert.New(t)

	pubkeyStr := "04e14529aa7c2a392dbe70f30f18cd0c76422d256fa413e151b87417d9232c41374985d8df6cedf084cf107c397ed658bd13dc2b31d4cbc3979c8684edb8b948bf"
	sigHex := "732292cff5543cd09efe0079e82edb53457ec9cef36d077b3fbc5dff62fa65f2105042810707505c4f98ed012661e60312c3c4e6b9eb815c64a2169c9c0ec7e8"
	msg := []byte("Gday, world!")

	sig := make([]byte, hex.DecodedLen(len(sigHex)))
	_, err := hex.Decode(sig, []byte(sigHex))
	if err != nil {
		t.Fatalf("Failed to decode signature: %s", err)
	}

	ok := VerifySignature(pubkeyStr, sig, msg)
	assert.True(ok)
}

func TestVerify(t *testing.T) {
	assert := assert.New(t)
	wallet, err := WalletFromPrivateKey("2053e3c0d239d12a554ef55895b89e5d044af7d09d8be9a8f6da22460f8260ca")
	if err != nil {
		t.Fatalf("Failed to create wallet: %s", err)
	}

	// Log pub key bytes
	t.Logf("Pubkey: %s", wallet.PubkeyStr())
	t.Logf("Pubkey: %s", wallet.PubkeyBytes())

	// Sign a message.
	msg := []byte("Gday, world!")
	sig, err := wallet.Sign(msg)
	if err != nil {
		t.Fatalf("Failed to sign message: %s", err)
	}

	t.Logf("Signature: %s", hex.EncodeToString(sig))

	// Verify the signature.
	pubkeyStr := wallet.PubkeyStr()
	ok := VerifySignature(pubkeyStr, sig, msg)

	assert.True(ok)
}

func TestRecreateWallet(t *testing.T) {
	assert := assert.New(t)

	wallet, err := WalletFromPrivateKey("2053e3c0d239d12a554ef55895b89e5d044af7d09d8be9a8f6da22460f8260ca")
	if err != nil {
		t.Fatalf("Failed to create wallet: %s", err)
	}

	assert.Equal(wallet.PubkeyStr(), "04e14529aa7c2a392dbe70f30f18cd0c76422d256fa413e151b87417d9232c41374985d8df6cedf084cf107c397ed658bd13dc2b31d4cbc3979c8684edb8b948bf")
	assert.Equal(wallet.Address(), "667a27f39789afa3975b44db3af94afd67292cbd9a18901d31ea25c666320b0c")
}
