package cmd

import (
	"github.com/liamzebedee/tinychain-go/core/nakamoto"
	"github.com/liamzebedee/tinychain-go/explorer"
	"github.com/urfave/cli/v2"

	"fmt"
	"os"
	"os/signal"
	"syscall"
)

func RunExplorer(cmdCtx *cli.Context) error {
	port := cmdCtx.Int("port")
	dbPath := cmdCtx.String("db")

	// DAG.
	networks := nakamoto.GetNetworks()
	dag, _, _ := newBlockdag(dbPath, networks["testnet1"].ConsensusConfig)

	// Handle process signals.
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c

		fmt.Println("Shutting down...")

		os.Exit(1)
	}()

	// Setup explorer.
	expl := explorer.NewBlockExplorerServer(&dag, port)
	expl.Start()

	return nil
}
