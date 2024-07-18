package core

import (
	"crypto/sha256"
)

func Hash(data []byte) [32]byte {
	return HashSHA2(data)
}

func HashSHA2(data []byte) [32]byte {
	return sha256.Sum256(data)
}

func HashPoseidon(data []byte) [32]byte {
	// TODO: implement.
	return [32]byte{}
}
