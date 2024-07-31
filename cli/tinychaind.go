package main

import (
	"log"
	"os"

	"github.com/liamzebedee/tinychain-go/cli/cmd"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:                 "tinychain",
		Usage:                "a simple blockchain implementation",
		EnableBashCompletion: true,
		Commands: []*cli.Command{
			{
				Name:   "node",
				Usage:  "runs the tinychain node",
				Action: cmd.RunNode,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "port",
						Usage: "The port to run the node on",
						Value: "8080",
					},
					&cli.StringFlag{
						Name:  "db",
						Usage: "The path to the tinychain database",
						Value: "tinychain.db",
						Required: true,
					},
					&cli.StringFlag{
						Name:  "peers",
						Usage: "A list of comma-separated peer URL's used to bootstrap connection to the network",
						Value: "",
					},
					&cli.BoolFlag{
						Name:  "miner",
						Usage: "Run the miner",
						Value: false,
					},
					&cli.BoolFlag{
						Name:  "explorer",
						Usage: "Run the block explorer on port 9000",
						Value: false,
					},
				},
			},
			{
				Name:   "explorer",
				Usage:  "runs the tinychain blockchain explorer web app",
				Action: cmd.RunExplorer,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "port",
						Usage: "The port to run the node on",
						Value: "9000",
					},
					&cli.StringFlag{
						Name:  "db",
						Usage: "The path to the tinychain database",
						Required: true,
					},
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
