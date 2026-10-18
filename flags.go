package main

import "github.com/urfave/cli/v2"

func flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:  "loglevel",
			Value: "info",
			Usage: "log level, range: [debug, info, warn, error, fatal]",
		},
		&cli.StringFlag{
			Name:  "logformat",
			Value: "text",
			Usage: "log format",
		},
		&cli.StringFlag{
			Name:  "logdir",
			Value: "./logs/",
			Usage: "log output directory",
		},
		&cli.BoolFlag{
			Name:  "report-func",
			Usage: "report function name",
			Value: false,
		},

		&cli.StringFlag{
			Name:  "config",
			Value: "config/spgw.yaml",
			Usage: "config file path",
		},
	}
}
