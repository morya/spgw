package main

import (
	"time"

	"github.com/urfave/cli/v2"
)

func sharedFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{Name: "spid", Required: true, Usage: "SP source address (6 bytes, e.g. 901234)"},
		&cli.StringFlag{Name: "password", Required: true, Usage: "shared secret with ISMG"},

		&cli.StringFlag{Name: "cmpp30-host", Value: "127.0.0.1:7891", Usage: "spgw iagent address (host:port)"},
		&cli.StringFlag{Name: "service-id", Value: "test", Usage: "ServiceID field"},
		&cli.StringFlag{Name: "fee-type", Value: "01", Usage: "FeeType field (2 bytes)"},
		&cli.StringFlag{Name: "src-id", Value: "10690000", Usage: "SrcID / display number"},
		&cli.StringFlag{Name: "sign", Value: "【测试桩】", Usage: "SMS signature, prepended to content"},
		&cli.StringFlag{Name: "content", Value: "测试短信", Usage: "SMS body"},

		&cli.DurationFlag{Name: "dial-timeout", Value: 5 * time.Second, Usage: "TCP dial timeout"},
	}
}

func benchFlags() []cli.Flag {
	return append(sharedFlags(),
		&cli.StringFlag{Name: "protocol", Value: "cmpp3.0", Usage: "cmpp2.0 / cmpp3.0"},

		&cli.IntFlag{Name: "rate", Value: 100, Usage: "send TPS"},
		&cli.IntFlag{Name: "total", Value: 1000, Usage: "total submit_req to send"},

		&cli.StringFlag{Name: "phone-from", Value: "13800000001", Usage: "first MSISDN (inclusive)"},
		&cli.StringFlag{Name: "phone-to", Value: "13800099999", Usage: "last MSISDN (inclusive)"},

		&cli.DurationFlag{Name: "drain-timeout", Value: 30 * time.Second, Usage: "wait window for late ack/report after sending finished"},

		&cli.StringFlag{Name: "result-file", Value: "./result.txt", Usage: "output statistics file"},
	)
}

func unittestFlags() []cli.Flag {
	return append(sharedFlags(),
		&cli.StringFlag{Name: "cmpp20-host", Value: "127.0.0.1:7890", Usage: "spgw iagent cmpp2.0 address (host:port)"},
		&cli.StringFlag{Name: "wrong-spid", Value: "nonexi", Usage: "non-existent SPID for negative test (6 bytes max)"},
		&cli.StringFlag{Name: "wrong-password", Value: "wrongpass", Usage: "wrong password for negative test"},
		&cli.StringFlag{Name: "test-result-file", Value: "./test_result.txt", Usage: "test result output file"},
	)
}
