// Tendermint/CometBFT consensus WIP.
// https://github.com/tendermint/tendermint/blob/main/spec/consensus/consensus.md#common-exit-conditions
// https://docs.tendermint.com/v0.34/introduction/what-is-tendermint.html#

/*
# The consensus engine runs a consensus algorithm called Tendermint.
# Tendermint is a byzantine fault-tolerant consensus algorithm.
# It consists of a validator set, where each validator is a node with a public key and some voting power.
# Transmuted into a blockchain, Tendermint is a proof-of-stake consensus protocol.
# Voting power corresponds to staked token balance.
# 
# [1]: https://github.com/tendermint/tendermint/blob/v0.34.x/spec/consensus/consensus.md
class TendermintConsensusEngine:
    def __init__(self, node):
        self.node = node

# vset - the validator set
# n - the number of validators
# VP(i) - voting power of validator i
# A(i) - accumulated priority for validator i
# P - total voting power of set
# avg - average of all validator priorities
# prop - proposer
def voting_power(i):
    return 0

# Select the proposer for the next epoch, from a dynamic validator set and 
# the history of past proposers (priority).
# [1]: https://github.com/tendermint/tendermint/blob/v0.34.x/spec/consensus/proposer-selection.md
def ProposerSelection(vset, priority):
    A = priority
    A2 = priority.copy()
    
    # P - total voting power of set
    P = sum(voting_power(i) for i in vset)

    # scale the priority values
    diff = max(A) - min(A)
    threshold = 2 * P
    if  diff > threshold:
        scale = diff/threshold
        for validator in vset:
            i = validator
            A2[i] = A[i]/scale

    # center priorities around zero
    avg = sum(A(i) for i in vset)/len(vset)
    for validator in vset:
        i = validator
        A2[i] -= avg

    # compute priorities and elect proposer
    for validator in vset:
        i = validator
        A2[i] += voting_power(i)
    
    prop = max(A)
    A2[prop] -= P
*/

package tendermint

import (
	"testing"
)

func TestTendermint(t *testing.T) {

}