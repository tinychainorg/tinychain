package nakamoto

import (
	"math"
)

// GetBlockReward returns the block reward in coins for a given block height.
// It uses the standard Bitcoin inflation curve.
func GetBlockReward(blockHeight int) uint64 {
	initialReward := 50.0
	halvingInterval := 210000

	// Calculate the number of halvings
	numHalvings := blockHeight / halvingInterval

	// Calculate the reward after the halvings
	reward_ := initialReward / math.Pow(2, float64(numHalvings))
	reward := uint64(reward_ * ONE_COIN)
	return reward
}

// ONE_COIN is the number of satoshis in one coin.
// Coin amounts are fixed-precision - they have 8 decimal places.
// 1 BTC = 1 * 10^8 = 100 000 000 sats
const ONE_COIN = 100_000_000
