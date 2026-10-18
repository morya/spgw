package link

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"time"

	protocol "github.com/hujm2023/go-sms-protocol"
	"github.com/hujm2023/go-sms-protocol/cmpp"
	"github.com/hujm2023/go-sms-protocol/cmpp/cmpp30"
	"github.com/hujm2023/go-sms-protocol/codec"

	"github.com/morya/spgw/internal/consts"
	"github.com/morya/spgw/spgwpb"
)

type CodecCMPP30 struct {
	framer codec.Codec
}

func NewCodecCMPP30() *CodecCMPP30 {
	return &CodecCMPP30{framer: codec.NewCMPPCodec()}
}

func (c *CodecCMPP30) Proto() string { return "cmpp30" }

func (c *CodecCMPP30) Decode(r codec.ConnReader) (msg, ack *spgwpb.Msg, err error) {
	data, err := c.framer.DecodeBlocked(r)
	if err != nil {
		return nil, nil, err
	}
	var pdu protocol.PDU
	pdu, err = cmpp30.DecodeCMPP30(data)
	if err != nil {
		return nil, nil, err
	}
	msg, ack, err = pduToMsgCmpp30(pdu, data)
	if err != nil {
		if ack != nil {
			return nil, ack, err
		}
		return nil, nil, err
	}
	return msg, ack, err
}

func (c *CodecCMPP30) Encode(msg *spgwpb.Msg) ([]byte, error) {
	pdu := msgToPduCmpp30(msg)
	if pdu == nil {
		return nil, fmt.Errorf("cmpp30: unsupported command %s", msg.Head.CommandId)
	}
	return pdu.IEncode()
}

func (c *CodecCMPP30) BuildBindReq(spid, pswd string, seqID uint32) *spgwpb.Msg {
	ts, tt := cmpp.GenConnectTimestamp(func() time.Time { return time.Now() })
	auth := cmpp.GenConnectAuth(spid, pswd, ts)
	return &spgwpb.Msg{
		Head: &spgwpb.Head{
			CommandId: spgwpb.CmdID_CmdBindReq,
			SeqId:     seqID,
		},
		BindAccountReq: &spgwpb.BindAccountReq{
			Spid:                spid,
			AuthenticatorSource: auth,
			Timestamp:           tt,
		},
	}
}

func IsValidTimestamp(timestamp uint32) bool {
	ts := cmpp.TimeStamp2Str(timestamp)
	if ts == "" {
		return false
	}
	// 强行补年份在时间戳字符串上
	year := time.Now().Format("2006")
	n, err := time.ParseInLocation("2006"+cmpp.ConnectTSFormat, year+ts, time.Local)
	if err != nil {
		return false
	}
	if time.Since(n) > time.Second*60*5 {
		return false
	}
	return true
}

//// ---------------------------------------------------------------------------
// pduToMsgCmpp30 分发器
// ---------------------------------------------------------------------------

func pduToMsgCmpp30(pdu protocol.PDU, data []byte) (*spgwpb.Msg, *spgwpb.Msg, error) {
	switch pdu.GetCommand().ToUint32() {

	case uint32(cmpp.CommandSubmit):
		return pduToMsg_submit30(pdu.(*cmpp30.Submit), data), nil, nil

	case uint32(cmpp.CommandSubmitResp):
		return pduToMsg_submitResp30(pdu.(*cmpp30.SubmitResp), data), nil, nil

	case uint32(cmpp.CommandDeliver):
		msg, err := pduToMsg_deliver30(pdu.(*cmpp30.Deliver), data)
		return msg, nil, err

	case uint32(cmpp.CommandDeliverResp):
		return pduToMsg_deliverResp30(pdu.(*cmpp30.DeliverResp), data), nil, nil

	case uint32(cmpp.CommandConnect):
		req := pdu.(*cmpp30.Connect)
		if len(req.SourceAddr) == 0 || len(req.AuthenticatorSource) == 0 || req.Timestamp == 0 {
			ack := spgwpb.NewMsg(spgwpb.CmdID_CmdBindAck, req.GetSequenceID())
			ack.Head.Status = consts.ErrCmppCodeMsgStruct
			return nil, ack, fmt.Errorf("connect source addr or authenticator source or timestamp is empty")
		}
		if req.Version != cmpp.Version30 {
			ack := spgwpb.NewMsg(spgwpb.CmdID_CmdBindAck, req.GetSequenceID())
			ack.Head.Status = consts.ErrCmppCodeVersionErr
			return nil, ack, fmt.Errorf("connect version %v not supported", req.Version)
		}
		if !IsValidTimestamp(req.Timestamp) {
			ack := spgwpb.NewMsg(spgwpb.CmdID_CmdBindAck, req.GetSequenceID())
			ack.Head.Status = consts.ErrCmppCodeAuthErr // 时间戳范围不对
			return nil, ack, fmt.Errorf("connect timestamp %v is out of range", req.Timestamp)
		}
		msg := pduToMsg_connect30(req, data)
		return msg, nil, nil

	case uint32(cmpp.CommandConnectResp):
		return pduToMsg_connectResp30(pdu.(*cmpp30.ConnectResp), data), nil, nil

	case uint32(cmpp.CommandActiveTest):
		msg, ack := pduToMsg_activeTest30(pdu)
		return msg, ack, nil

	case uint32(cmpp.CommandActiveTestResp):
		return pduToMsg_activeTestResp30(pdu), nil, nil

	case uint32(cmpp.CommandTerminate):
		msg, ack := pduToMsg_terminate30(pdu)
		return msg, ack, nil

	default:
		// 未识别的请求命令（bit31=0）：生成空应答并立即回复，seq 一一对应
		// 未识别的应答命令（bit31=1）：直接透传，无需回复
		cmd := pdu.GetCommand().ToUint32()
		seq := pdu.GetSequenceID()
		if cmd&0x80000000 == 0 {
			resp := pdu.GenEmptyResponse()
			if resp != nil {
				raw, _ := resp.IEncode()
				return nil, &spgwpb.Msg{
					Head: &spgwpb.Head{
						CommandId: spgwpb.CmdID_CmdUnknown,
						SeqId:     resp.GetSequenceID(),
						Status:    0,
						RecvTime:  time.Now().UnixMilli(),
					},
					RawMsg: raw,
				}, nil
			}
			return nil, nil, nil
		}
		return &spgwpb.Msg{
			Head: &spgwpb.Head{
				CommandId: spgwpb.CmdID_CmdUnknown,
				SeqId:     seq,
				RecvTime:  time.Now().UnixMilli(),
			},
			RawMsg: data,
		}, nil, nil
	}
}

// ---------------------------------------------------------------------------
// msgToPduCmpp30 分发器
// ---------------------------------------------------------------------------

func msgToPduCmpp30(msg *spgwpb.Msg) protocol.PDU {
	switch msg.Head.CommandId {

	case spgwpb.CmdID_CmdSubmitReq:
		return msgToPdu_submitReq30(msg.SubmitReq, msg.GetHead().SeqId)

	case spgwpb.CmdID_CmdSubmitAck:
		return msgToPdu_submitAck30(msg.SubmitAck, msg.GetHead().SeqId)

	case spgwpb.CmdID_CmdDeliverReq:
		return msgToPdu_deliverReq30(msg.DeliverReq, msg.GetHead().SeqId)

	case spgwpb.CmdID_CmdDeliverAck:
		return msgToPdu_deliverAck30(msg, msg.GetHead().SeqId)

	case spgwpb.CmdID_CmdReportReq:
		return msgToPdu_reportReq30(msg.ReportReq, msg.GetHead().SeqId)

	case spgwpb.CmdID_CmdReportAck:
		return msgToPdu_reportAck30(msg.ReportAck, msg.GetHead().SeqId)

	case spgwpb.CmdID_CmdBindReq:
		return msgToPdu_bindReq30(msg.BindAccountReq, msg.Head.SeqId)

	case spgwpb.CmdID_CmdBindAck:
		return msgToPdu_bindAck30(msg.BindAccountRsp, msg.GetHead().Status, msg.Head.SeqId)

	case spgwpb.CmdID_CmdActiveTestReq:
		return msgToPdu_activeTestReq30(msg.GetHead().SeqId)

	case spgwpb.CmdID_CmdActiveTestAck:
		return msgToPdu_activeTestAck30(msg.GetHead().SeqId)

	default:
		return nil
	}
}

// ---------------------------------------------------------------------------
// pduToMsg 辅助函数（每种 PDU 类型一个独立函数）
// ---------------------------------------------------------------------------

func pduToMsg_submit30(v *cmpp30.Submit, data []byte) *spgwpb.Msg {
	return &spgwpb.Msg{
		Head: &spgwpb.Head{
			CommandId: spgwpb.CmdID_CmdSubmitReq,
			SeqId:     v.GetSequenceID(),
			RecvTime:  time.Now().UnixMilli(),
		},
		RawMsg: data,
		SubmitReq: &spgwpb.SubmitReq{
			MsgId:              v.MsgID,
			PkTotal:            uint32(v.PkTotal),
			PkNumber:           uint32(v.PkNumber),
			RegisteredDelivery: uint32(v.RegisteredDelivery),
			MsgLevel:           uint32(v.MsgLevel),
			ServiceId:          v.ServiceID,
			FeeUserType:        uint32(v.FeeUserType),
			FeeTerminalId:      v.FeeTerminalID,
			FeeTerminalType:    uint32(v.FeeTerminalType),
			TpPid:              uint32(v.TpPID),
			TpUdhi:             uint32(v.TpUDHI),
			MsgFmt:             uint32(v.MsgFmt),
			MsgSrc:             v.MsgSrc,
			FeeType:            v.FeeType,
			FeeCode:            v.FeeCode,
			ValidTime:          v.ValiDTime,
			AtTime:             v.AtTime,
			SrcId:              v.SrcID,
			DestUsrTl:          uint32(v.DestUsrTL),
			DestTerminalId:     v.DestTerminalID,
			DestTerminalType:   uint32(v.DestTerminalType),
			MsgLength:          uint32(v.MsgLength),
			MsgContent:         v.MsgContent,
			LinkId:             v.LinkID,
		},
	}
}

func pduToMsg_submitResp30(v *cmpp30.SubmitResp, data []byte) *spgwpb.Msg {
	return &spgwpb.Msg{
		Head: &spgwpb.Head{
			CommandId: spgwpb.CmdID_CmdSubmitAck,
			SeqId:     v.GetSequenceID(),
			Status:    v.Result,
			RecvTime:  time.Now().UnixMilli(),
		},
		RawMsg: data,
		SubmitAck: &spgwpb.SubmitAck{
			MsgId:  v.MsgID,
			Result: v.Result,
		},
	}
}

func pduToMsg_deliver30(v *cmpp30.Deliver, data []byte) (*spgwpb.Msg, error) {
	if v.RegisteredDeliver == 1 {
		var rpt cmpp.SubPduDeliveryContent
		if err := rpt.IDecode(v.MsgContent); err != nil {
			return nil, fmt.Errorf("decode Report Content failed: %w", err)
		}
		return &spgwpb.Msg{
			Head: &spgwpb.Head{
				CommandId: spgwpb.CmdID_CmdReportReq,
				SeqId:     v.GetSequenceID(),
				RecvTime:  time.Now().UnixMilli(),
			},
			RawMsg: data,
			ReportReq: &spgwpb.ReportReq{
				DeliverBase: &spgwpb.DeliverBase{
					MsgId:              v.MsgID,
					DestId:             v.DestID,
					ServiceId:          v.ServiceID,
					TpPid:              uint32(v.TpPID),
					TpUdhi:             uint32(v.TpUDHI),
					MsgFmt:             uint32(v.MsgFmt),
					SrcTerminalId:      v.SrcTerminalID,
					RegisteredDelivery: uint32(v.RegisteredDeliver),
				},
				MsgId:          rpt.MsgID,
				Stat:           rpt.Stat,
				SubmitTime:     rpt.SubmitTime,
				DoneTime:       rpt.DoneTime,
				DestTerminalId: rpt.DestTerminalID,
				SmscSequence:   rpt.SMSCSequence,
			},
		}, nil
	}
	return &spgwpb.Msg{
		Head: &spgwpb.Head{
			CommandId: spgwpb.CmdID_CmdDeliverReq,
			SeqId:     v.GetSequenceID(),
			RecvTime:  time.Now().UnixMilli(),
		},
		RawMsg: data,
		DeliverReq: &spgwpb.DeliverReq{
			DeliverBase: &spgwpb.DeliverBase{
				MsgId:              v.MsgID,
				DestId:             v.DestID,
				ServiceId:          v.ServiceID,
				TpPid:              uint32(v.TpPID),
				TpUdhi:             uint32(v.TpUDHI),
				MsgFmt:             uint32(v.MsgFmt),
				SrcTerminalId:      v.SrcTerminalID,
				SrcTerminalType:    uint32(v.SrcTerminalType),
				RegisteredDelivery: uint32(v.RegisteredDeliver),
			},
			MsgLength:  uint32(v.MsgLength),
			MsgContent: v.MsgContent,
			LinkId:     v.LinkID,
		},
	}, nil
}

func pduToMsg_deliverResp30(v *cmpp30.DeliverResp, data []byte) *spgwpb.Msg {
	return &spgwpb.Msg{
		Head: &spgwpb.Head{
			CommandId: spgwpb.CmdID_CmdDeliverAck,
			SeqId:     v.GetSequenceID(),
			RecvTime:  time.Now().UnixMilli(),
		},
		RawMsg: data,
		DeliverAck: &spgwpb.DeliverAck{
			MsgId:  v.MsgID,
			Result: v.Result,
		},
	}
}

func pduToMsg_connect30(v *cmpp30.Connect, data []byte) *spgwpb.Msg {
	req := &spgwpb.Msg{
		Head: &spgwpb.Head{
			CommandId: spgwpb.CmdID_CmdBindReq,
			SeqId:     v.GetSequenceID(),
			RecvTime:  time.Now().UnixMilli(),
		},
		RawMsg: data,
		BindAccountReq: &spgwpb.BindAccountReq{
			Spid:                v.SourceAddr,
			AuthenticatorSource: v.AuthenticatorSource[:],
			Timestamp:           v.Timestamp,
		},
	}
	return req
}

func pduToMsg_connectResp30(v *cmpp30.ConnectResp, data []byte) *spgwpb.Msg {
	return &spgwpb.Msg{
		Head: &spgwpb.Head{
			CommandId: spgwpb.CmdID_CmdBindAck,
			SeqId:     v.GetSequenceID(),
			Status:    v.Status,
			RecvTime:  time.Now().UnixMilli(),
		},
		RawMsg: data,
		BindAccountRsp: &spgwpb.BindAccountRsp{
			AuthenticatorIsmg: []byte(v.AuthenticatorISMG[:]),
		},
	}
}

func pduToMsg_activeTest30(pdu protocol.PDU) (msg, ack *spgwpb.Msg) {
	resp := pdu.GenEmptyResponse()
	raw, _ := resp.IEncode()

	msg = &spgwpb.Msg{
		Head: &spgwpb.Head{
			CommandId: spgwpb.CmdID_CmdActiveTestReq,
			SeqId:     pdu.GetSequenceID(),
			RecvTime:  time.Now().UnixMilli(),
		},
	}
	ack = &spgwpb.Msg{
		Head: &spgwpb.Head{
			CommandId: spgwpb.CmdID_CmdActiveTestAck,
			SeqId:     resp.GetSequenceID(),
			Status:    0,
			RecvTime:  time.Now().UnixMilli(),
		},
		RawMsg: raw,
	}
	return msg, ack
}

func pduToMsg_activeTestResp30(pdu protocol.PDU) *spgwpb.Msg {
	return &spgwpb.Msg{
		Head: &spgwpb.Head{
			CommandId: spgwpb.CmdID_CmdActiveTestAck,
			SeqId:     pdu.GetSequenceID(),
			RecvTime:  time.Now().UnixMilli(),
		},
	}
}

func pduToMsg_terminate30(pdu protocol.PDU) (msg, ack *spgwpb.Msg) {
	resp := pdu.GenEmptyResponse()
	raw, _ := resp.IEncode()

	msg = &spgwpb.Msg{
		Head: &spgwpb.Head{
			CommandId: spgwpb.CmdID_CmdTerminateReq,
			SeqId:     pdu.GetSequenceID(),
			RecvTime:  time.Now().UnixMilli(),
		},
	}
	ack = &spgwpb.Msg{
		Head: &spgwpb.Head{
			CommandId: spgwpb.CmdID_CmdTerminateAck,
			SeqId:     resp.GetSequenceID(),
			Status:    0,
			RecvTime:  time.Now().UnixMilli(),
		},
		RawMsg: raw,
	}
	return msg, ack
}

// ---------------------------------------------------------------------------
// msgToPdu 辅助函数（每种 Msg 类型一个独立函数）
// ---------------------------------------------------------------------------

func msgToPdu_submitReq30(sr *spgwpb.SubmitReq, seqID uint32) *cmpp30.Submit {
	return &cmpp30.Submit{
		Header: cmpp.Header{
			SequenceID: seqID,
			CommandID:  cmpp.CommandSubmit,
		},
		MsgID:              sr.MsgId,
		PkTotal:            uint8(sr.PkTotal),
		PkNumber:           uint8(sr.PkNumber),
		RegisteredDelivery: uint8(sr.RegisteredDelivery),
		MsgLevel:           uint8(sr.MsgLevel),
		ServiceID:          sr.ServiceId,
		FeeUserType:        uint8(sr.FeeUserType),
		FeeTerminalID:      sr.FeeTerminalId,
		FeeTerminalType:    uint8(sr.FeeTerminalType),
		TpPID:              uint8(sr.TpPid),
		TpUDHI:             uint8(sr.TpUdhi),
		MsgFmt:             uint8(sr.MsgFmt),
		MsgSrc:             sr.MsgSrc,
		FeeType:            sr.FeeType,
		FeeCode:            sr.FeeCode,
		ValiDTime:          sr.ValidTime,
		AtTime:             sr.AtTime,
		SrcID:              sr.SrcId,
		DestUsrTL:          uint8(sr.DestUsrTl),
		DestTerminalID:     sr.DestTerminalId,
		DestTerminalType:   uint8(sr.DestTerminalType),
		MsgLength:          uint8(sr.MsgLength),
		MsgContent:         sr.MsgContent,
		LinkID:             sr.LinkId,
	}
}

func msgToPdu_submitAck30(sa *spgwpb.SubmitAck, seqID uint32) *cmpp30.SubmitResp {
	return &cmpp30.SubmitResp{
		Header: cmpp.Header{
			SequenceID: seqID,
			CommandID:  cmpp.CommandSubmitResp,
		},
		MsgID:  sa.MsgId,
		Result: sa.Result,
	}
}

func msgToPdu_deliverReq30(dr *spgwpb.DeliverReq, seqID uint32) *cmpp30.Deliver {
	return &cmpp30.Deliver{
		Header: cmpp.Header{
			SequenceID: seqID,
			CommandID:  cmpp.CommandDeliver,
		},

		MsgID:             dr.DeliverBase.MsgId,
		DestID:            dr.DeliverBase.DestId,
		ServiceID:         dr.DeliverBase.ServiceId,
		TpPID:             uint8(dr.DeliverBase.TpPid),
		TpUDHI:            uint8(dr.DeliverBase.TpUdhi),
		MsgFmt:            uint8(dr.DeliverBase.MsgFmt),
		SrcTerminalID:     dr.DeliverBase.SrcTerminalId,
		SrcTerminalType:   uint8(dr.DeliverBase.SrcTerminalType),
		RegisteredDeliver: uint8(dr.DeliverBase.RegisteredDelivery),
		MsgLength:         uint8(dr.MsgLength),
		MsgContent:        dr.MsgContent,
		LinkID:            dr.LinkId,
	}
}

func msgToPdu_deliverAck30(m *spgwpb.Msg, seqID uint32) *cmpp30.DeliverResp {
	p := &cmpp30.DeliverResp{
		Header: cmpp.Header{
			SequenceID: seqID,
			CommandID:  cmpp.CommandDeliverResp,
		},
	}

	if m.DeliverAck != nil {
		p.MsgID = m.DeliverAck.MsgId
		p.Result = m.DeliverAck.Result
		return p
	}
	if m.ReportAck != nil {
		p.MsgID = m.ReportAck.MsgId
		p.Result = m.ReportAck.Result
		return p
	}
	return p
}

func msgToPdu_reportReq30(rr *spgwpb.ReportReq, seqID uint32) *cmpp30.Deliver {
	rpt := cmpp.SubPduDeliveryContent{
		MsgID:          rr.MsgId,
		Stat:           rr.Stat,
		SubmitTime:     rr.SubmitTime,
		DoneTime:       rr.DoneTime,
		DestTerminalID: rr.DestTerminalId,
		SMSCSequence:   rr.SmscSequence,
	}
	content, _ := rpt.IEncode()
	return &cmpp30.Deliver{
		Header: cmpp.Header{
			SequenceID: seqID,
			CommandID:  cmpp.CommandDeliver,
		},
		MsgID:             rr.DeliverBase.MsgId,
		DestID:            rr.DeliverBase.DestId,
		ServiceID:         rr.DeliverBase.ServiceId,
		TpPID:             uint8(rr.DeliverBase.TpPid),
		TpUDHI:            uint8(rr.DeliverBase.TpUdhi),
		MsgFmt:            uint8(rr.DeliverBase.MsgFmt),
		SrcTerminalID:     rr.DeliverBase.SrcTerminalId,
		SrcTerminalType:   uint8(rr.DeliverBase.SrcTerminalType),
		RegisteredDeliver: uint8(rr.DeliverBase.RegisteredDelivery),

		MsgLength:  uint8(len(content)),
		MsgContent: content,
	}
}

func msgToPdu_reportAck30(ra *spgwpb.ReportAck, seqID uint32) *cmpp30.DeliverResp {
	return &cmpp30.DeliverResp{
		Header: cmpp.Header{
			SequenceID: seqID,
			CommandID:  cmpp.CommandDeliverResp,
		},
		MsgID:  ra.MsgId,
		Result: ra.Result,
	}
}

func msgToPdu_bindReq30(br *spgwpb.BindAccountReq, seqID uint32) *cmpp30.Connect {
	v := &cmpp30.Connect{
		Header: cmpp.Header{
			CommandID:  cmpp.CommandConnect,
			SequenceID: seqID,
		},
		SourceAddr: br.Spid,
		// AuthenticatorSource: br.Auth,
		Version:   cmpp.Version30,
		Timestamp: br.Timestamp,
	}
	copy(v.AuthenticatorSource[:], br.AuthenticatorSource[:])
	return v
}

func msgToPdu_bindAck30(br *spgwpb.BindAccountRsp, status uint32, seqID uint32) *cmpp30.ConnectResp {
	v3 := &cmpp30.ConnectResp{
		Header: cmpp.Header{
			SequenceID: seqID,
			CommandID:  cmpp.CommandConnectResp,
		},
	}
	v3.Status = status
	if br != nil {
		buff := bytes.NewBuffer(nil)
		binary.Write(buff, binary.BigEndian, status)
		copy(v3.AuthenticatorISMG[:], cmpp.GenConnectRespAuthISMG(buff.Bytes(), br.AuthenticatorSource, br.Password))
	}
	v3.Version = cmpp.Version30
	return v3
}

func msgToPdu_activeTestReq30(seqID uint32) *cmpp30.ActiveTest {
	return &cmpp30.ActiveTest{
		Header: cmpp.Header{
			CommandID:  cmpp.CommandActiveTest,
			SequenceID: seqID,
		},
	}
}

func msgToPdu_activeTestAck30(seqID uint32) *cmpp30.ActiveTestResp {
	return &cmpp30.ActiveTestResp{
		Header: cmpp.Header{
			CommandID:  cmpp.CommandActiveTestResp,
			SequenceID: seqID,
		},
	}
}
