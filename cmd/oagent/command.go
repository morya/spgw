package oagent

import (
	"github.com/urfave/cli/v2"
)

var Command = &cli.Command{
	Name:    "oagent",
	Aliases: []string{"o", "oa"},
	Usage:   "oagent",
	Before:  before,
	Flags: []cli.Flag{
		&cli.IntFlag{
			Name:  "http-port",
			Usage: "http listen port",
			Value: 8083,
		},
	},
	Action: action,
}
