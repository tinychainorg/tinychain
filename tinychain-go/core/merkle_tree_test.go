package core

import (
	"testing"
	"crypto/sha256"
	"github.com/stretchr/testify/assert"
	"fmt"
	"encoding/hex"
)

func TestMerkleTreeAccumulate(t *testing.T) {
	assert := assert.New(t)

	// Generate a test set of 100 items with indexed keys of H(i).
	items := make([][]byte, 100)

	for i := 0; i < 100; i++ {
		hash := sha256.Sum256([]byte(fmt.Sprintf("%d", i)))
		items = append(items, hash[:])
	}

	// Compute the expected root hash.
	expected := ComputeMerkleHash(items)
	t.Logf("Expected root hash: %x", expected)

	expectedStr := hex.EncodeToString(expected[:])
	assert.Equal(expectedStr, "9d88c165d938bbc80c02fc856ddca3028f30b11fabff4cce14280742b031d5b6")
}