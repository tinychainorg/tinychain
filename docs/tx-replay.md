Transaction replay
==================

Transaction replay in a blockchain is an attack where someone who has sent a transaction like "send 100 BTC from Alice to Bob", can later have that transaction sent again (replayed) by a malicious attacker. The blockchain code needs to prevent transactions being replayed; the code needs to ensure transactions are processed only ONCE.

There are a few approaches.

## Bitcoin.

Bitcoin uses a UXTO model, whereby transactions consume one unspent TX output (UXTO). The bitcoin node maintains an index of UXTO's at all times, and if a UXTO is missing from this set, that means the transaction has been processed.

## Ethereum.

Ethereum uses an account-based model. Each transaction sent from an account increments the nonce, and only one transaction-per-nonce value can be processed. This prevents replay though doesn't prevent malicious nodes from reordering txs with the same nonce.

## Solana.

https://blog.fordefi.com/warping-time-for-solana-defi-reconciling-expiring-nonces-and-institutional-policy-controls#:~:text=Solana%20protects%20against%20replay%20attacks,been%20included%20in%20the%20blockchain.

> Solana protects against replay attacks by disallowing inclusion of any transaction that is identical to a transaction that has already been included in the blockchain. The way this is implemented is that every transaction must include a hash of a recent block, where a block is considered recent if it is at most 151 slots old, which corresponds to 1–2 minutes. Each validator maintains a list of all the recent transactions, and when it receives a new transaction, it checks that: (i) the transaction includes a recent blockhash; and (ii) the transaction is not identical to any of the recent transactions. The validator only includes the transaction if it satisfies both conditions.

