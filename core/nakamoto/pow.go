package nakamoto

import (
	"math/big"
	"fmt"
)

func VerifyPOW(blockhash [32]byte, target big.Int) bool {
	fmt.Printf("VerifyPOW target: %s\n", target.String())

	hash := new(big.Int).SetBytes(blockhash[:])
	return hash.Cmp(&target) == -1
}

func SolvePOW(b RawBlock, startNonce big.Int, target big.Int, maxIterations uint64) (big.Int, error) {
	fmt.Printf("SolvePOW target: %s\n", target.String())

	block := b
	nonce := startNonce
	var i uint64 = 0

	for {
		i++
		
		// Exit if iterations is reached.
		if maxIterations < i {
			return big.Int{}, fmt.Errorf("Solution not found in %d iterations", maxIterations)
		}

		// Increment nonce.
		nonce.Add(&nonce, big.NewInt(1))
		block.SetNonce(nonce)

		// Hash.
		h := block.Hash()
		hash := new(big.Int).SetBytes(h[:])

		// Check solution: hash < target.
		if hash.Cmp(&target) == -1 {
			fmt.Printf("Solved in %d iterations\n", i)
			fmt.Printf("Hash: %x\n", hash.String())
			fmt.Printf("Nonce: %s\n", nonce.String())
			return nonce, nil
		}
	}
}

func RecomputeDifficulty(epochStart uint64, epochEnd uint64, currDifficulty big.Int, targetEpochLengthMillis uint64, epochLengthBlocks uint64, height uint64) (big.Int) {
	// Compute the epoch duration.
	epochDuration := epochEnd - epochStart
	
	// Special case: clamp the epoch duration so it is at least 1.
	if epochDuration == 0 {
		epochDuration = 1
	}

	epochIndex := height / epochLengthBlocks

	fmt.Printf("epoch i=%d start_time=%d end_time=%d duration=%d \n", epochIndex, epochStart, epochEnd, epochDuration)

	// Compute the target epoch length.
	targetEpochLength := targetEpochLengthMillis * epochLengthBlocks

	// Rescale the difficulty.
	// difficulty = epoch.difficulty * (epoch.duration / target_epoch_length)
	newDifficulty := new(big.Int)
	newDifficulty.Mul(
		&currDifficulty, 
		big.NewInt(int64(epochDuration)),
	)
	newDifficulty.Div(
		newDifficulty, 
		big.NewInt(int64(targetEpochLength)),
	)

	fmt.Printf("New difficulty: %x\n", newDifficulty.String())
	
	return *newDifficulty
}