package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
)

type Config struct {
	Listen string

	Accounts map[string]string

	CallbackURL           string
	CallbackDelay         time.Duration
	CallbackFailRate      float64
	CallbackRetry         int
	CallbackRetryInterval time.Duration
	CallbackBatchSize     int

	BatchMax        int
	RequestTimeout  time.Duration
	ShutdownTimeout time.Duration
}

func parseConfig(c *cli.Context) (*Config, error) {
	cfg := &Config{
		Listen:                c.String("listen"),
		Accounts:              map[string]string{"100001": "88888888"},
		CallbackURL:           c.String("callback-url"),
		CallbackDelay:         c.Duration("callback-delay"),
		CallbackFailRate:      c.Float64("callback-fail-rate"),
		CallbackRetry:         c.Int("callback-retry"),
		CallbackRetryInterval: c.Duration("callback-retry-interval"),
		CallbackBatchSize:     c.Int("callback-batch-size"),
		BatchMax:              c.Int("batch-max"),
		RequestTimeout:        c.Duration("request-timeout"),
		ShutdownTimeout:       c.Duration("shutdown-timeout"),
	}

	for _, raw := range c.StringSlice("account") {
		parts := strings.SplitN(raw, ":", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return nil, fmt.Errorf("invalid --account %q, want 'appId:apiKey'", raw)
		}
		cfg.Accounts[parts[0]] = parts[1]
	}

	if cfg.CallbackFailRate < 0 || cfg.CallbackFailRate > 1 {
		return nil, fmt.Errorf("callback-fail-rate must be in [0.0, 1.0]")
	}
	if cfg.BatchMax <= 0 {
		return nil, fmt.Errorf("batch-max must be > 0")
	}
	if cfg.CallbackBatchSize <= 0 {
		return nil, fmt.Errorf("callback-batch-size must be > 0")
	}
	return cfg, nil
}
