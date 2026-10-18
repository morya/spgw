package main

import (
	"time"

	"github.com/urfave/cli/v2"
)

func flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{Name: "listen", Value: ":9900", Usage: "HTTP listen address"},

		&cli.StringSliceFlag{
			Name:  "account",
			Usage: "register account, format 'appId:apiKey', repeatable",
		},

		&cli.StringFlag{Name: "callback-url", Value: "", Usage: "client report callback URL; empty = log only"},
		&cli.DurationFlag{Name: "callback-delay", Value: 3 * time.Second, Usage: "delay before pushing report"},
		&cli.Float64Flag{Name: "callback-fail-rate", Value: 0.1, Usage: "probability to mark report as UNDELIV (0.0~1.0)"},
		&cli.IntFlag{Name: "callback-retry", Value: 3, Usage: "max retry on callback HTTP failure"},
		&cli.DurationFlag{Name: "callback-retry-interval", Value: 5 * time.Second, Usage: "interval between callback retries"},
		&cli.IntFlag{Name: "callback-batch-size", Value: 50, Usage: "max reports per callback HTTP request"},

		&cli.IntFlag{Name: "batch-max", Value: 1000, Usage: "max phones per batchSend request"},
		&cli.DurationFlag{Name: "request-timeout", Value: 30 * time.Second, Usage: "max time to read request body"},
		&cli.DurationFlag{Name: "shutdown-timeout", Value: 30 * time.Second, Usage: "graceful shutdown drain window"},

		&cli.StringFlag{Name: "log-level", Value: "info", Usage: "debug / info / warn / error"},
	}
}
