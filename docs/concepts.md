Concept index.
==============

This is an index of the concepts you will encounter throughout a Nakamoto blockchain.

 - Hash.
 - Signature.
 - Hashcash puzzle. 
 - Work.
 - Difficulty.
    - Target.
    - Retargeting based on average solution rate.
 - Epoch.
   - Canonical epoch ID's.
 - Transaction.
   - Canonical transaction ID's.
 - Accounts.
 - Block.
 - Canonical encoding for hashing (envelope). Excluding the signature.
 - Network encoding (raw) vs. client storage (raw + metadata).
 - DAG's, trees.
   - Tip. 
   - Height / Depth.
   - Heaviest chain / accumulated work.
 - Sync. 
   - History vs. state sync.
   - Greedy search. Window sizing.
   - Bit sets.
   - Parallel downloads / chunking.
 - State machine.
   - State transitions. Coinbase and regular transactions.
   - State leaf.
   - Fee payment. 
   - Integer overflows.
   - State construction is map-reduce. Map transactions to state leaves through state machine transition function. Apply the effects (state leaves) to current state. Map each block, reduce to new state, feed into next block.
 - Mempool.
   - Bundles.
   - Builders.
   - Fee optimization.
   - Blockspace auctions.
   - Fee rate.
 - Mining.
   - Block template - header and bundle.
   - Nonce.
   - Puzzles. Guess. Solution.
   - Hashrate.
 - Peers and networking.
   - Messages.
   - RPC.
   - Gossip.
   - Heartbeat.
 - Node.
   - The kernel of the blockchain, the main loop, event handler, the wiring of the DAG, the miner and the peer.
 - Light clients ("SPV").
   - Merkle trees. Merklization.
   - Block headers. Block bodies.
 - Proof-of-work consensus.
   - Longest chain rule, game theory why this is relevant during sync.

