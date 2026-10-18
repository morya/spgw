package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/urfave/cli/v2"
)

type Config struct {
	Host     string
	Protocol string

	Spid     string
	Password string

	Rate  int
	Total int

	PhoneFrom uint64
	PhoneTo   uint64

	Sign      string
	Content   string
	ServiceID string
	FeeType   string
	SrcID     string

	DrainTimeout time.Duration
	DialTimeout  time.Duration

	ResultFile string
}

func parseConfig(c *cli.Context) (*Config, error) {
	cfg := &Config{
		Host:         c.String("cmpp30-host"),
		Protocol:     strings.ToLower(c.String("protocol")),
		Spid:         c.String("spid"),
		Password:     c.String("password"),
		Rate:         c.Int("rate"),
		Total:        c.Int("total"),
		Sign:         c.String("sign"),
		Content:      c.String("content"),
		ServiceID:    c.String("service-id"),
		FeeType:      c.String("fee-type"),
		SrcID:        c.String("src-id"),
		DrainTimeout: c.Duration("drain-timeout"),
		DialTimeout:  c.Duration("dial-timeout"),
		ResultFile:   c.String("result-file"),
	}

	switch cfg.Protocol {
	case "cmpp2.0", "cmpp20":
		cfg.Protocol = "cmpp2.0"
	case "cmpp3.0", "cmpp30":
		cfg.Protocol = "cmpp3.0"
	default:
		return nil, fmt.Errorf("unknown protocol %q (want cmpp2.0 / cmpp3.0)", cfg.Protocol)
	}

	from, err := strconv.ParseUint(c.String("phone-from"), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("phone-from invalid: %w", err)
	}
	to, err := strconv.ParseUint(c.String("phone-to"), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("phone-to invalid: %w", err)
	}
	if from > to {
		return nil, fmt.Errorf("phone-from > phone-to")
	}
	cfg.PhoneFrom = from
	cfg.PhoneTo = to

	if cfg.Rate <= 0 {
		return nil, fmt.Errorf("rate must be > 0")
	}
	if cfg.Total <= 0 {
		return nil, fmt.Errorf("total must be > 0")
	}
	if cfg.Spid == "" || cfg.Password == "" {
		return nil, fmt.Errorf("spid / password required")
	}
	return cfg, nil
}

func (c *Config) IsCMPP30() bool { return c.Protocol == "cmpp3.0" }
