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

// 1 BTC = 1 * 10^8
// BTC amounts are fixed-precision - they have 8 decimal places.
// ie. 1 BTC = 100 000 000 sats
const ONE_COIN = 100_000_000
