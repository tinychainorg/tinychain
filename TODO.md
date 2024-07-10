## To Do.

Work breakdown:

- Refactor block DAG
    - Headers tip
    - Bodies tip
    - Ingest headers
    - ingest Bodies
    - ingest block
- rename: GetPath
- getHeadersTip
- GetBodiesTip
- OnNewBodiesTip/OnNewHeadersTip


GetBlockByHash - return block but with no txs (this is default behaviour anyways)
GetBlockTransactions - error if not synced block
GetLatestTip - do we want the tip for the fully synced chain or only the light chain?
GetLongestChainHashList - this is fine
GetRawBlockDataByHash - this would error if we don't have full block
HasBlock - this might fail
IngestBlock - this is fine

State machine has to filter - 
    it only maps blocks with txs
    how do we know if we have the txs for a block? 
    1. annotate block with .synced / .full
    2. two data structures - block_headers block_bodies. link to block. filter on blocks where we have the body. modify methods that would need full body. but then miner should only mine on the tip that has full body.

    type BlockDAG
        func NewBlockDAGFromDB(db *sql.DB, stateMachine StateMachineInterface, consensus ConsensusConfig) (BlockDAG, error)
        func (dag *BlockDAG) GetBlockByHash(hash [32]byte) (*Block, error)
        func (dag *BlockDAG) GetBlockTransactions(hash [32]byte) (*[]Transaction, error)
        func (dag *BlockDAG) GetEpochForBlockHash(blockhash [32]byte) (*Epoch, error)
        func (dag *BlockDAG) GetLatestTip() (Block, error)
        func (dag *BlockDAG) GetLongestChainHashList(startHash [32]byte, depthFromTip uint64) ([][32]byte, error)
        func (dag *BlockDAG) GetRawBlockDataByHash(hash [32]byte) ([]byte, error)
        func (dag *BlockDAG) HasBlock(hash [32]byte) bool
        func (dag *BlockDAG) IngestBlock(raw RawBlock) error

at its core, focus on implementing the data structure correctly and all the methods will follow from that

a dag tracks blocks
    blocks can be full with data (block header + block body) or just light with the header (block header)

what does the light chain need to do? 
- download headers
- get the current tip of the heaviest chain (headers only or full blocks)
    - ingest headers do not recompute state
- recompute difficulty

does it need to handle block metadata?
	Height          uint64       - yes
	Epoch           string       - yes
	Work            big.Int      - yes
	SizeBytes       uint64       - yes
	Hash            [32]byte     - yes
	AccumulatedWork big.Int      - yes

otherwise it doesn't need to track anything else??

once we have the headers, then we just download the blocks in order and ingest them


I think it's easy enough for the DAG to track just headers to provide a lightweight view of the DAG
and then when our full node gets out of sync
it resyncs the light graph
and then syncs the full blocks into a raw_block cache
and calls blockdag to ingest them one-by-one

the headers will be duplicated? but that's okay?
idk it's a lot of headers.



I think a light dag will work
it just means storing twice the number of block headers??
which is fine since block headers are quite small
the other risk is getting out of sync
it's worthwhile not complexifying the implementation


what about get tip though? 


hash functions:

- pow
- verify pow
- merkle
- block hash