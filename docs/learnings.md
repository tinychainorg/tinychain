# Learnings.

 * ECDSA signatures are 65 bytes
 * simpler to store raw blocks since we want to retransmit them later
 * syncing is hard. best embed the parenttotalwork so we can choose the latest tip easily.
 * concurrency
   * I encountered a latent error when a test for one peer receiving a heartbeat from another would pass individually, but fail when run as part of the test suite. Turns out, the server would fail silently if there was another peer listening on the port. Correcting for this, you can do two things - (1) get a random port from the system for the tests, and (2) fail if the port is already bound.
 * the build initially worked on my machine, but failed when compiling for ubuntu. This was because SQLite dependency requires cross-compiler for C to be installed.
 * there are lots of things I didn't anticipate:
   * signature caching
   * `blocks_transactions` table. Originally thought transactions.block was a one-to-one. Derp, it's many-to-many.
 * to check if a block is fully synced, we need to 
