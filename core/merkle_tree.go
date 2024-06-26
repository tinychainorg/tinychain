package core

import (
	"crypto/sha256"
)

func ComputeMerkleHash(items [][]byte) [32]byte {
	if len(items) == 0 {
		return [32]byte{}
	}
	if len(items) == 1 {
		return sha256.Sum256(items[0])
	}
	mid := len(items) / 2
	left := ComputeMerkleHash(items[:mid])
	right := ComputeMerkleHash(items[mid:])
	return sha256.Sum256(append(left[:], right[:]...))
}

