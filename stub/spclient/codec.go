package main

import (
	"fmt"
	"time"

	sms "github.com/hujm2023/go-sms-protocol"
	"github.com/hujm2023/go-sms-protocol/cmpp"
	"github.com/hujm2023/go-sms-protocol/cmpp/cmpp20"
	"github.com/hujm2023/go-sms-protocol/cmpp/cmpp30"
	"golang.org/x/text/encoding/simplifiedchinese"
)

type SubmitParams struct {
	SeqID     uint32
	MsgID     uint64
	Phone     string
	Content   string
	ServiceID string
	FeeType   string
	SrcID     string
	MsgSrc    string
}

type Codec interface {
	BuildConnect(seqID uint32, srcAddr, password string) ([]byte, error)
	BuildSubmit(p SubmitParams) ([]byte, error)
	BuildActiveTestResp(seqID uint32) ([]byte, error)
	BuildDeliverResp(seqID uint32, msgID uint64) ([]byte, error)
	BuildTerminate(seqID uint32) ([]byte, error)
	Decode(data []byte) (sms.PDU, error)
}

func NewCodec(proto string) (Codec, error) {
	switch proto {
	case "cmpp2.0":
		return &codecCMPP20{}, nil
	case "cmpp3.0":
		return &codecCMPP30{}, nil
	default:
		return nil, fmt.Errorf("unsupported protocol %q", proto)
	}
}

func encodeGB18030(s string) ([]byte, error) {
	enc := simplifiedchinese.GB18030.NewEncoder()
	return enc.Bytes([]byte(s))
}

func packBody(content string) ([]byte, uint8, error) {
	b, err := encodeGB18030(content)
	if err != nil {
		return nil, 0, err
	}
	if len(b) > 252 {
		b = b[:252]
	}
	return b, uint8(len(b)), nil
}

type codecCMPP30 struct{}

func (c *codecCMPP30) BuildConnect(seqID uint32, srcAddr, password string) ([]byte, error) {
	ts, tt := cmpp.GenConnectTimestamp(time.Now)
	auth := cmpp.GenConnectAuth(srcAddr, password, ts)
	pdu := &cmpp30.Connect{
		Header: cmpp.Header{
			CommandID:  cmpp.CommandConnect,
			SequenceID: seqID,
		},
		SourceAddr: srcAddr,
		Version:    cmpp.Version30,
		Timestamp:  tt,
	}
	copy(pdu.AuthenticatorSource[:], auth[:])

	return pdu.IEncode()
}

func (c *codecCMPP30) BuildSubmit(p SubmitParams) ([]byte, error) {
	body, length, err := packBody(p.Content)
	if err != nil {
		return nil, err
	}
	pdu := &cmpp30.Submit{
		Header: cmpp.Header{
			CommandID:  cmpp.CommandSubmit,
			SequenceID: p.SeqID,
		},
		MsgID:              p.MsgID,
		PkTotal:            1,
		PkNumber:           1,
		RegisteredDelivery: 1,
		MsgLevel:           1,
		ServiceID:          p.ServiceID,
		FeeUserType:        2,
		FeeTerminalID:      "",
		FeeTerminalType:    0,
		TpPID:              0,
		TpUDHI:             0,
		MsgFmt:             15,
		MsgSrc:             p.MsgSrc,
		FeeType:            p.FeeType,
		FeeCode:            "000000",
		ValiDTime:          "",
		AtTime:             "",
		SrcID:              p.SrcID,
		DestUsrTL:          1,
		DestTerminalID:     []string{p.Phone},
		DestTerminalType:   0,
		MsgLength:          length,
		MsgContent:         body,
		LinkID:             "",
	}
	return pdu.IEncode()
}

func (c *codecCMPP30) BuildActiveTestResp(seqID uint32) ([]byte, error) {
	pdu := &cmpp30.ActiveTestResp{
		Header: cmpp.Header{
			CommandID:  cmpp.CommandActiveTestResp,
			SequenceID: seqID,
		},
	}
	return pdu.IEncode()
}

func (c *codecCMPP30) BuildDeliverResp(seqID uint32, msgID uint64) ([]byte, error) {
	pdu := &cmpp30.DeliverResp{
		Header: cmpp.Header{
			CommandID:  cmpp.CommandDeliverResp,
			SequenceID: seqID,
		},
		MsgID:  msgID,
		Result: 0,
	}
	return pdu.IEncode()
}

func (c *codecCMPP30) BuildTerminate(seqID uint32) ([]byte, error) {
	pdu := &cmpp30.Terminate{
		Header: cmpp.Header{
			CommandID:  cmpp.CommandTerminate,
			SequenceID: seqID,
		},
	}
	return pdu.IEncode()
}

func (c *codecCMPP30) Decode(data []byte) (sms.PDU, error) {
	return cmpp30.DecodeCMPP30(data)
}

type codecCMPP20 struct{}

func (c *codecCMPP20) BuildConnect(seqID uint32, srcAddr, password string) ([]byte, error) {
	ts, tt := cmpp.GenConnectTimestamp(time.Now)
	auth := string(cmpp.GenConnectAuth(srcAddr, password, ts))
	pdu := &cmpp20.PduConnect{
		Header: cmpp.Header{
			CommandID:  cmpp.CommandConnect,
			SequenceID: seqID,
		},
		SourceAddr: srcAddr,
		// 	AuthenticatorSource: auth,
		Version:   cmpp.Version20,
		Timestamp: tt,
	}
	copy(pdu.AuthenticatorSource[:], auth[:])

	return pdu.IEncode()
}

func (c *codecCMPP20) BuildSubmit(p SubmitParams) ([]byte, error) {
	body, length, err := packBody(p.Content)
	if err != nil {
		return nil, err
	}
	pdu := &cmpp20.PduSubmit{
		Header: cmpp.Header{
			CommandID:  cmpp.CommandSubmit,
			SequenceID: p.SeqID,
		},
		MsgID:              p.MsgID,
		PkTotal:            1,
		PkNumber:           1,
		RegisteredDelivery: 1,
		MsgLevel:           1,
		ServiceID:          p.ServiceID,
		FeeUserType:        2,
		FeeTerminalID:      "",
		TpPID:              0,
		TpUDHI:             0,
		MsgFmt:             15,
		MsgSrc:             p.MsgSrc,
		FeeType:            p.FeeType,
		FeeCode:            "000000",
		ValIDTime:          "",
		AtTime:             "",
		SrcID:              p.SrcID,
		DestUsrTL:          1,
		DestTerminalID:     []string{p.Phone},
		MsgLength:          length,
		MsgContent:         body,
		Reserve:            "",
	}
	return pdu.IEncode()
}

func (c *codecCMPP20) BuildActiveTestResp(seqID uint32) ([]byte, error) {
	pdu := &cmpp20.PduActiveTestResp{
		Header: cmpp.Header{
			CommandID:  cmpp.CommandActiveTestResp,
			SequenceID: seqID,
		},
	}
	return pdu.IEncode()
}

func (c *codecCMPP20) BuildDeliverResp(seqID uint32, msgID uint64) ([]byte, error) {
	pdu := &cmpp20.PduDeliverResp{
		Header: cmpp.Header{
			CommandID:  cmpp.CommandDeliverResp,
			SequenceID: seqID,
		},
		MsgID:  msgID,
		Result: 0,
	}
	return pdu.IEncode()
}

func (c *codecCMPP20) BuildTerminate(seqID uint32) ([]byte, error) {
	pdu := &cmpp20.PduTerminate{
		Header: cmpp.Header{
			CommandID:  cmpp.CommandTerminate,
			SequenceID: seqID,
		},
	}
	return pdu.IEncode()
}

func (c *codecCMPP20) Decode(data []byte) (sms.PDU, error) {
	return cmpp20.DecodeCMPP20(data)
}

// ---------------------------------------------------------------------------
// 测试用异常场景构造函数
// ---------------------------------------------------------------------------

// ConnectParams 支持自定义参数的 connect 构造，用于测试异常场景
type ConnectParams struct {
	SeqID     uint32
	SrcAddr   string
	Password  string // AuthBytes 为 nil 时用此计算鉴权码
	Version   uint8  // 0 表示用协议默认版本
	AuthBytes []byte // 非 nil 时直接使用，不通过 password 计算
	Timestamp uint32 // 0 + AuthBytes!=nil 时保持 0；0 + AuthBytes==nil 时用当前时间
	Proto     string // "cmpp2.0" / "cmpp3.0"
}

// BuildConnectEx 构造自定义参数的 connect PDU
func BuildConnectEx(p ConnectParams) ([]byte, error) {
	proto := p.Proto
	if proto == "" {
		proto = "cmpp3.0"
	}

	version := p.Version
	ts := p.Timestamp
	var authStr string

	if p.AuthBytes != nil {
		authStr = string(p.AuthBytes)
	} else {
		var tsStr string
		if ts == 0 {
			tsStr, ts = cmpp.GenConnectTimestamp(time.Now)
		} else {
			tsStr = cmpp.TimeStamp2Str(ts)
		}
		authStr = string(cmpp.GenConnectAuth(p.SrcAddr, p.Password, tsStr))
	}

	if version == 0 {
		if proto == "cmpp3.0" {
			version = cmpp.Version30
		} else {
			version = cmpp.Version20
		}
	}

	switch proto {
	case "cmpp2.0":
		pdu := &cmpp20.PduConnect{
			Header:     cmpp.Header{CommandID: cmpp.CommandConnect, SequenceID: p.SeqID},
			SourceAddr: p.SrcAddr,
			// AuthenticatorSource: authStr,
			Version:   version,
			Timestamp: ts,
		}
		copy(pdu.AuthenticatorSource[:], authStr[:])
		return pdu.IEncode()
	case "cmpp3.0":
		pdu := &cmpp30.Connect{
			Header:     cmpp.Header{CommandID: cmpp.CommandConnect, SequenceID: p.SeqID},
			SourceAddr: p.SrcAddr,
			// AuthenticatorSource: authStr,
			Version:   version,
			Timestamp: ts,
		}
		copy(pdu.AuthenticatorSource[:], authStr[:])
		return pdu.IEncode()
	default:
		return nil, fmt.Errorf("unsupported protocol %q", proto)
	}
}

// SubmitExParams 支持群发和长消息的 submit 构造
type SubmitExParams struct {
	SeqID     uint32
	Phones    []string
	Content   string
	ServiceID string
	FeeType   string
	SrcID     string
	MsgSrc    string
	PkTotal   uint8
	PkNumber  uint8
	Proto     string
}

// BuildSubmitEx 构造自定义参数的 submit PDU
func BuildSubmitEx(p SubmitExParams) ([]byte, error) {
	proto := p.Proto
	if proto == "" {
		proto = "cmpp3.0"
	}
	if len(p.Phones) == 0 {
		return nil, fmt.Errorf("phones is empty")
	}
	pkTotal := p.PkTotal
	if pkTotal == 0 {
		pkTotal = 1
	}
	pkNumber := p.PkNumber
	if pkNumber == 0 {
		pkNumber = 1
	}
	body, length, err := packBody(p.Content)
	if err != nil {
		return nil, err
	}
	destUsrTL := uint8(len(p.Phones))

	switch proto {
	case "cmpp2.0":
		pdu := &cmpp20.PduSubmit{
			Header:             cmpp.Header{CommandID: cmpp.CommandSubmit, SequenceID: p.SeqID},
			PkTotal:            pkTotal,
			PkNumber:           pkNumber,
			RegisteredDelivery: 1,
			MsgLevel:           1,
			ServiceID:          p.ServiceID,
			FeeUserType:        2,
			TpPID:              0,
			TpUDHI:             0,
			MsgFmt:             15,
			MsgSrc:             p.MsgSrc,
			FeeType:            p.FeeType,
			FeeCode:            "000000",
			SrcID:              p.SrcID,
			DestUsrTL:          destUsrTL,
			DestTerminalID:     p.Phones,
			MsgLength:          length,
			MsgContent:         body,
		}
		return pdu.IEncode()
	case "cmpp3.0":
		pdu := &cmpp30.Submit{
			Header:             cmpp.Header{CommandID: cmpp.CommandSubmit, SequenceID: p.SeqID},
			PkTotal:            pkTotal,
			PkNumber:           pkNumber,
			RegisteredDelivery: 1,
			MsgLevel:           1,
			ServiceID:          p.ServiceID,
			FeeUserType:        2,
			FeeTerminalType:    0,
			TpPID:              0,
			TpUDHI:             0,
			MsgFmt:             15,
			MsgSrc:             p.MsgSrc,
			FeeType:            p.FeeType,
			FeeCode:            "000000",
			SrcID:              p.SrcID,
			DestUsrTL:          destUsrTL,
			DestTerminalID:     p.Phones,
			DestTerminalType:   0,
			MsgLength:          length,
			MsgContent:         body,
			LinkID:             "",
		}
		return pdu.IEncode()
	default:
		return nil, fmt.Errorf("unsupported protocol %q", proto)
	}
}
