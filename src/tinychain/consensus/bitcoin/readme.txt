The bitcoin consensus protocol (or Nakamoto consensus) is split into these components:

block.py      - the base block data structure, transmitted over the network, referred to as a "raw block".

dag.py        - the block DAG, a directed acyclic graph of blocks in the chain. 
                each block has a height which refers to its depth in a chain of blocks.
                each block has an accumulated work, which represents the cumulative hashpower to mine up to this block.

consensus.py  - the core of the bitcoin/nakamoto consensus engine.
                includes functionality for mining blocks, calcuating difficulty epochs, adding new blocks to the DAG.

node.py       - the core of the bitcoin node.
                includes a miner thread, to mine new blocks
                includes a sync thread, to get the latest blockchain "tip" from other nodes, and download their blocks.