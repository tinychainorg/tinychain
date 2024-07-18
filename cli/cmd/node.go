package cmd

import (
	"github.com/liamzebedee/tinychain-go/core"
	"github.com/liamzebedee/tinychain-go/core/nakamoto"
	"github.com/urfave/cli/v2"

	"database/sql"
	"encoding/hex"
	"fmt"
	"math/big"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

type MockStateMachine struct{}

func newMockStateMachine() *MockStateMachine {
	return &MockStateMachine{}
}
func (m *MockStateMachine) VerifyTx(tx nakamoto.RawTransaction) error {
	return nil
}

func newBlockdag(dbPath string) (nakamoto.BlockDAG, nakamoto.ConsensusConfig, *sql.DB) {
	// TODO validate connection string.
	db, err := nakamoto.OpenDB(dbPath)
	if err != nil {
		panic(err)
	}
	_, err = db.Exec("PRAGMA journal_mode = WAL;")
	if err != nil {
		panic(err)
	}

	stateMachine := newMockStateMachine()

	genesis_difficulty := new(big.Int)
	genesis_difficulty.SetString("0fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", 16)

	// https://serhack.me/articles/story-behind-alternative-genesis-block-bitcoin/ ;)
	genesisBlockHash_, err := hex.DecodeString("000006b15d1327d67e971d1de9116bd60a3a01556c91b6ebaa416ebc0cfaa646")
	if err != nil {
		panic(err)
	}
	genesisBlockHash := [32]byte{}
	copy(genesisBlockHash[:], genesisBlockHash_)

	conf := nakamoto.ConsensusConfig{
		EpochLengthBlocks:       10,
		TargetEpochLengthMillis: 1000 * 60 * 5, // 5 minutes
		GenesisDifficulty:       *genesis_difficulty,
		GenesisParentBlockHash:  genesisBlockHash,
		MaxBlockSizeBytes:       2 * 1024 * 1024, // 2MB
	}

	blockdag, err := nakamoto.NewBlockDAGFromDB(db, stateMachine, conf)
	if err != nil {
		panic(err)
	}

	return blockdag, conf, db
}

func RunNode(cmdCtx *cli.Context) error {
	port := cmdCtx.String("port")
	dbPath := cmdCtx.String("db")
	bootstrapPeers := cmdCtx.String("peers")
	runMiner := cmdCtx.Bool("miner")

	// DAG.
	dag, _, _ := newBlockdag(dbPath)

	// Miner.
	minerWallet, err := core.CreateRandomWallet()
	if err != nil {
		return err
	}

	miner := nakamoto.NewMiner(dag, minerWallet)

	// Peer.
	peer := nakamoto.NewPeerCore(nakamoto.NewPeerConfig("0.0.0.0", port, []string{}))

	// Create the node.
	node := nakamoto.NewNode(&dag, miner, peer)

	// Handle process signals.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c

		fmt.Println("Shutting down...")
		node.Shutdown()

		os.Exit(1)
	}()

	// Bootstrap the node.
	if bootstrapPeers != "" {
		peerAddresses := []string{}
		// Split the comma-separated list of peer addresses.
		peerlist := strings.Split(bootstrapPeers, ",")
		for _, peerAddress := range peerlist {
			// Validate URL.
			_, err := url.ParseRequestURI(peerAddress)
			if err != nil {
				return fmt.Errorf("Invalid peer address: %s", peerAddress)
			}
			peerAddresses = append(peerAddresses, peerAddress)
		}

		node.Peer.Bootstrap(peerAddresses)
	}

	if runMiner {
		go node.Miner.Start(-1)
	}

	node.Start()
	return nil
}
