package core

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"math/big"
	"fmt"
)

type Wallet struct {
	prvkey *ecdsa.PrivateKey
}

func (w *Wallet) Pubkey() *ecdsa.PublicKey {
	return &w.prvkey.PublicKey
}

func (w *Wallet) PubkeyBytes() [65]byte {
	pubkey := w.Pubkey()
	
	// 	The length of the buffer returned by elliptic.Marshal depends on the elliptic curve used. For the NIST P-256 curve (also known as elliptic.P256()), the buffer will be 65 bytes long. This includes:

	// 1 byte for the format prefix (0x04 for uncompressed)
	// 32 bytes for the X coordinate
	// 32 bytes for the Y coordinate

	buf := elliptic.Marshal(pubkey.Curve, pubkey.X, pubkey.Y)
	var pubkeyBytes [65]byte
	copy(pubkeyBytes[:], buf)
	return pubkeyBytes
}

func (w *Wallet) PubkeyStr() string {
	pubkey := w.PubkeyBytes()
	return hex.EncodeToString(pubkey[:])
}

func (w *Wallet) PrvkeyStr() string {
	return hex.EncodeToString(w.prvkey.D.Bytes())
}

func (w *Wallet) Address() string {
	pubkeyStr := w.PubkeyStr()
	firstHash := sha256.Sum256([]byte(pubkeyStr))
	secondHash := sha256.Sum256(firstHash[:])
	return hex.EncodeToString(secondHash[:])
}

func CreateRandomWallet() (*Wallet, error) {
	prvkey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	return &Wallet{prvkey: prvkey}, nil
}

func WalletFromPrivateKey(privateKeyHex string) (*Wallet, error) {
	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return nil, err
	}
	prvkey := new(ecdsa.PrivateKey)
	prvkey.D = new(big.Int).SetBytes(privateKeyBytes)
	prvkey.PublicKey.Curve = elliptic.P256()
	prvkey.PublicKey.X, prvkey.PublicKey.Y = prvkey.PublicKey.Curve.ScalarBaseMult(privateKeyBytes)
	return &Wallet{prvkey: prvkey}, nil
}

func padBytes(src []byte, length int) []byte {
    if len(src) >= length {
        return src
    }
    padding := make([]byte, length-len(src))
    return append(padding, src...)
}

func (w *Wallet) Sign(msg []byte) ([]byte, error) {
	hash := sha256.Sum256(msg)
	r, s, err := ecdsa.Sign(rand.Reader, w.prvkey, hash[:])
	if err != nil {
		panic(err)
		return nil, err
	}
	// Ensure r and s are padded to 32 bytes
	rBytes := padBytes(r.Bytes(), 32)
	sBytes := padBytes(s.Bytes(), 32)

	// Concatenate r and s to form the signature
	signature := append(rBytes, sBytes...)
	
	return signature, nil
}

func VerifySignature(pubkeyStr string, sig, msg []byte) bool {
	if len(sig) != 64 {
		fmt.Printf("Invalid signature length: %s\n", len(sig)) // TODO
		return false
	}
	if len(pubkeyStr) != 130 {
		panic("Invalid public key") // TODO
		return false
	}

	pubkeyBytes, err := hex.DecodeString(pubkeyStr)
	if err != nil {
		panic(err)
		return false
	}

	x, y := elliptic.Unmarshal(elliptic.P256(), pubkeyBytes)
	if x == nil {
		panic("Invalid public key") // TODO
		return false
	}
	pubkey := &ecdsa.PublicKey{Curve: elliptic.P256(), X: x, Y: y}

	hash := sha256.Sum256(msg)
	r := new(big.Int).SetBytes(sig[:len(sig)/2])
	s := new(big.Int).SetBytes(sig[len(sig)/2:])

	// fmt.Println(r.Sign() <= 0 || s.Sign() <= 0)
	fmt.Printf("%s\n", pubkeyStr)
	fmt.Printf("r=%s s=%s\n", r.String(), s.String())

	return ecdsa.Verify(pubkey, hash[:], r, s)
}

