package nakamoto

import (
	"math"
)

// GetBlockReward returns the block reward for a given block height.
// It uses the standard Bitcoin inflation curve.
func GetBlockReward(blockHeight int) float64 {
	initialReward := 50.0
	halvingInterval := 210000

	// Calculate the number of halvings
	numHalvings := blockHeight / halvingInterval

	// Calculate the reward after the halvings
	reward := initialReward / math.Pow(2, float64(numHalvings))
	return reward
}
