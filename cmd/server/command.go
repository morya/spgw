package server

import (
	"github.com/urfave/cli/v2"
)

var Command = &cli.Command{
	Name:    "server",
	Aliases: []string{"s"},
	Usage:   "server",
	Before:  before,
	Flags: []cli.Flag{
		&cli.IntFlag{
			Name:  "http-port",
			Usage: "http listen port",
			Value: 8082,
		},
	},
	Action: action,
}
