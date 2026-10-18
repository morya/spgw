package iagent

import (
	"github.com/urfave/cli/v2"
)

var Command = &cli.Command{
	Name:    "iagent",
	Aliases: []string{"i", "ia"},
	Usage:   "iagent",
	Flags: []cli.Flag{
		&cli.IntFlag{
			Name:        "cmpp30-port",
			Usage:       "cmpp30 listen port",
			Value:       7891,
			Destination: &portCmpp30,
		},
		&cli.IntFlag{
			Name:        "cmpp20-port",
			Usage:       "cmpp20 listen port",
			Value:       7890,
			Destination: &portCmpp20,
		},
		&cli.IntFlag{
			Name:  "http-port",
			Usage: "http listen port",
			Value: 8081,
		},
	},
	Before: before,
	Action: action,
}
