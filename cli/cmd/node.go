package cmd

import (
	"github.com/liamzebedee/tinychain-go/core"
	"github.com/liamzebedee/tinychain-go/core/nakamoto"
	"github.com/liamzebedee/tinychain-go/explorer"
	"github.com/urfave/cli/v2"

	"database/sql"
	"fmt"
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

func newBlockdag(dbPath string, conf nakamoto.ConsensusConfig) (nakamoto.BlockDAG, nakamoto.ConsensusConfig, *sql.DB) {
	// TODO validate connection string.
	fmt.Println("database path: ", dbPath)
	db, err := nakamoto.OpenDB(dbPath)
	if err != nil {
		panic(err)
	}
	_, err = db.Exec("PRAGMA journal_mode = WAL;")
	if err != nil {
		panic(err)
	}

	stateMachine := newMockStateMachine()

	blockdag, err := nakamoto.NewBlockDAGFromDB(db, stateMachine, conf)
	if err != nil {
		panic(err)
	}

	return blockdag, conf, db
}

func getMinerWallet(db *sql.DB) (*core.Wallet, error) {
	walletsStore, err := nakamoto.LoadDataStore[nakamoto.WalletsStore](db, "wallets")
	if err != nil {
		return nil, err
	}
	if 0 == len(walletsStore.Wallets) {
		wallet, err := core.CreateRandomWallet()
		if err != nil {
			return nil, err
		}

		walletsStore.Wallets = append(walletsStore.Wallets, nakamoto.UserWallet{
			Label:            "miner",
			PrivateKeyString: wallet.PrvkeyStr(),
		})

		err = nakamoto.SaveDataStore(db, "wallets", *walletsStore)
		if err != nil {
			return nil, err
		}
	}
	minerWallet, err := core.WalletFromPrivateKey(walletsStore.Wallets[0].PrivateKeyString)
	if err != nil {
		return nil, err
	}
	return minerWallet, nil
}

func RunNode(cmdCtx *cli.Context) error {
	port := cmdCtx.String("port")
	dbPath := cmdCtx.String("db")
	bootstrapPeers := cmdCtx.String("peers")
	runMiner := cmdCtx.Bool("miner")
	runExplorer := cmdCtx.Bool("explorer")
	network := cmdCtx.String("network")
	graffitiTag := cmdCtx.String("miner-tag")

	if network == "" {
		network = "testnet1"
	}

	// DAG.
	networks := nakamoto.GetNetworks()
	net, ok := networks[network]
	if !ok {
		availableNetworks := []string{}
		for k := range networks {
			availableNetworks = append(availableNetworks, k)
		}
		fmt.Printf("Available networks: %s\n", strings.Join(availableNetworks, ", "))
		return fmt.Errorf("Unknown network: %s", network)
	}
	dag, _, db := newBlockdag(dbPath, net.ConsensusConfig)

	// Miner.
	minerWallet, err := getMinerWallet(db)
	if err != nil {
		return err
	}
	fmt.Printf("Miner wallet: %x\n", minerWallet.PubkeyBytes())
	miner := nakamoto.NewMiner(dag, minerWallet)
	miner.GraffitiTag = nakamoto.StringToBytes32(graffitiTag)

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

	if runExplorer {
		expl := explorer.NewBlockExplorerServer(&dag, 9000)
		go expl.Start()
	}

	node.Start()
	return nil
}
