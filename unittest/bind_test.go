package unittest

import (
	"encoding/hex"
	"testing"
	"time"

	"github.com/hujm2023/go-sms-protocol/cmpp"
	"github.com/hujm2023/go-sms-protocol/cmpp/cmpp30"
)

func buildConnectMsg1(srcAddr, pswd string) *cmpp30.Connect {
	ts, tt := cmpp.GenConnectTimestamp(func() time.Time {
		t := time.Unix(1781020800, 0)
		return t
	})
	auth := cmpp.GenConnectAuth(srcAddr, pswd, ts)

	m := &cmpp30.Connect{
		Header: cmpp.Header{
			CommandID: cmpp.CommandConnect,
		},
		SourceAddr: srcAddr,
		// AuthenticatorSource: string(auth),
		Version:   cmpp.Version30,
		Timestamp: tt,
	}
	copy(m.AuthenticatorSource[:], auth[:])
	return m
}

func build1() []byte {
	req := buildConnectMsg1("8888", "changeme")
	data, _ := req.IEncode()
	return data
}

func TestBind(t *testing.T) {
	data1 := build1()

	t.Logf("data1 length = %d", len(data1))
	t.Logf("data1 = %s", hex.Dump(data1))
}
