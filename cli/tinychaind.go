package main

import (
	"github.com/urfave/cli/v2"
	"github.com/liamzebedee/tinychain-go/cli/cmd"
	"log"
	"os"
)

func main() {
	app := &cli.App{
        Name: "tinychain",
        Usage: "a simple blockchain implementation",
        EnableBashCompletion: true,
        Commands: []*cli.Command{
            {
                Name:    "node",
                Usage:   "runs the tinychain node",
                Action: cmd.RunNode,
                Flags: []cli.Flag{
                    &cli.StringFlag{
                        Name:    "port",
                        Usage:   "The port to run the node on",
                        Value: "8080",
                    },
                    &cli.StringFlag{
                        Name:    "db",
                        Usage:   "The path to the tinychain database",
                        Value: "tinychain.db",
                    },
                },        
            },
        },
    }

    if err := app.Run(os.Args); err != nil {
        log.Fatal(err)
    }
}