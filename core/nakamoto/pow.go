package nakamoto

import (
	"fmt"
	"math/big"
)

var powLogger = NewLogger("pow", "")

// Verifies a proof-of-work solution.
func VerifyPOW(blockhash [32]byte, target big.Int) bool {
	powLogger.Printf("VerifyPOW target: %s\n", target.String())

	hash := new(big.Int).SetBytes(blockhash[:])
	return hash.Cmp(&target) == -1
}

// Solves a proof-of-work puzzle.
func SolvePOW(b RawBlock, startNonce big.Int, target big.Int, maxIterations uint64) (big.Int, error) {
	powLogger.Printf("SolvePOW target: %s\n", target.String())

	block := b
	nonce := startNonce
	var i uint64 = 0

	for {
		i++

		// Exit if iterations is reached.
		if maxIterations != 0 && maxIterations < i {
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
			powLogger.Printf("Solved in %d iterations\n", i)
			powLogger.Printf("Hash: %x\n", hash.String())
			powLogger.Printf("Nonce: %s\n", nonce.String())
			return nonce, nil
		}
	}
}

// Recomputes the difficulty for the next epoch.
func RecomputeDifficulty(epochStart uint64, epochEnd uint64, currDifficulty big.Int, targetEpochLengthMillis uint64, epochLengthBlocks uint64, height uint64) big.Int {
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

	powLogger.Printf("New difficulty: %x\n", newDifficulty.String())

	return *newDifficulty
}

// Calculates the work of a solution.
func CalculateWork(solution big.Int) *big.Int {
	// In Bitcoin POW, the work is defined as:
	// work = 2^256 / (diff_target + 1)
	// We take a difference approach. Note the invariant: solution < diff_target. For the puzzle to be solved, it must always satisfy this condition.
	// While estimating the work as a function of the difficulty target does work,
	// it is more precise to estimate the work of the individual guess (solution).
	work := big.NewInt(2).Exp(big.NewInt(2), big.NewInt(256), nil)
	solutionPtr := &solution
	work.Div(work, big.NewInt(0).Add(solutionPtr, big.NewInt(1)))
	return work
}
