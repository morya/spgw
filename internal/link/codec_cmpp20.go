package link

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"time"

	protocol "github.com/hujm2023/go-sms-protocol"
	"github.com/hujm2023/go-sms-protocol/cmpp"
	"github.com/hujm2023/go-sms-protocol/cmpp/cmpp20"
	"github.com/hujm2023/go-sms-protocol/codec"
	"go.uber.org/zap"

	"github.com/morya/spgw/internal/consts"
	"github.com/morya/spgw/spgwpb"
)

type CodecCMPP20 struct {
	framer codec.Codec
}

func NewCodecCMPP20() *CodecCMPP20 {
	return &CodecCMPP20{framer: codec.NewCMPPCodec()}
}

func (c *CodecCMPP20) Proto() string { return "cmpp20" }

func (c *CodecCMPP20) Decode(r codec.ConnReader) (msg, ack *spgwpb.Msg, err error) {
	data, err := c.framer.DecodeBlocked(r)
	if err != nil {
		return nil, nil, err
	}
	var pdu protocol.PDU
	pdu, err = cmpp20.DecodeCMPP20(data)
	if err != nil {
		return nil, nil, err
	}
	msg, ack, err = pduToMsgCmpp20(pdu, data)
	return msg, ack, err
}

func (c *CodecCMPP20) Encode(msg *spgwpb.Msg) ([]byte, error) {
	pdu := msgToPduCmpp20(msg)
	if pdu == nil {
		return nil, fmt.Errorf("cmpp20: unsupported command %s", msg.Head.CommandId)
	}
	return pdu.IEncode()
}

func (c *CodecCMPP20) BuildBindReq(spid, pswd string, seqID uint32) *spgwpb.Msg {
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

// ---------------------------------------------------------------------------
// msgToPduCmpp20 分发器
// ---------------------------------------------------------------------------

func msgToPduCmpp20(msg *spgwpb.Msg) protocol.PDU {
	switch msg.Head.CommandId {

	case spgwpb.CmdID_CmdSubmitReq:
		return msgToPdu_submitReq20(msg.SubmitReq, msg.GetHead().SeqId)

	case spgwpb.CmdID_CmdSubmitAck:
		return msgToPdu_submitAck20(msg.SubmitAck, msg.GetHead().SeqId)

	case spgwpb.CmdID_CmdDeliverReq:
		return msgToPdu_deliverReq20(msg.DeliverReq, msg.GetHead().SeqId)

	case spgwpb.CmdID_CmdDeliverAck:
		return msgToPdu_deliverAck20(msg, msg.GetHead().SeqId)

	case spgwpb.CmdID_CmdReportReq:
		return msgToPdu_reportReq20(msg.ReportReq, msg.GetHead().SeqId)

	case spgwpb.CmdID_CmdReportAck:
		return msgToPdu_reportAck20(msg.ReportAck, msg.GetHead().SeqId)

	case spgwpb.CmdID_CmdBindReq:
		return msgToPdu_bindReq20(msg.BindAccountReq, msg.Head.SeqId)

	case spgwpb.CmdID_CmdBindAck:
		return msgToPdu_bindAck20(msg.BindAccountRsp, msg.GetHead().Status, msg.Head.SeqId)

	case spgwpb.CmdID_CmdActiveTestReq:
		return msgToPdu_activeTestReq20(msg.GetHead().SeqId)

	case spgwpb.CmdID_CmdActiveTestAck:
		return msgToPdu_activeTestAck20(msg.GetHead().SeqId)

	default:
		return nil
	}
}

// ---------------------------------------------------------------------------
// pduToMsg 辅助函数（每种 PDU 类型一个独立函数）
// ---------------------------------------------------------------------------

func pduToMsg_submit20(v *cmpp20.PduSubmit, data []byte) *spgwpb.Msg {
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
			TpPid:              uint32(v.TpPID),
			TpUdhi:             uint32(v.TpUDHI),
			MsgFmt:             uint32(v.MsgFmt),
			MsgSrc:             v.MsgSrc,
			FeeType:            v.FeeType,
			FeeCode:            v.FeeCode,
			ValidTime:          v.ValIDTime,
			AtTime:             v.AtTime,
			SrcId:              v.SrcID,
			DestUsrTl:          uint32(v.DestUsrTL),
			DestTerminalId:     v.DestTerminalID,
			MsgLength:          uint32(v.MsgLength),
			MsgContent:         v.MsgContent,
		},
	}
}

func pduToMsg_submitResp20(v *cmpp20.PduSubmitResp, data []byte) *spgwpb.Msg {
	return &spgwpb.Msg{
		Head: &spgwpb.Head{
			CommandId: spgwpb.CmdID_CmdSubmitAck,
			SeqId:     v.GetSequenceID(),
			Status:    uint32(v.Result),
			RecvTime:  time.Now().UnixMilli(),
		},
		RawMsg: data,
		SubmitAck: &spgwpb.SubmitAck{
			MsgId:  v.MsgID,
			Result: uint32(v.Result),
		},
	}
}

func pduToMsg_deliver20(v *cmpp20.PduDeliver, data []byte) (*spgwpb.Msg, error) {
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
				RegisteredDelivery: uint32(v.RegisteredDeliver),
			},
			MsgLength:  uint32(v.MsgLength),
			MsgContent: v.MsgContent,
			LinkId:     v.Reserved,
		},
	}, nil
}

func pduToMsg_deliverResp20(v *cmpp20.PduDeliverResp, data []byte) *spgwpb.Msg {
	return &spgwpb.Msg{
		Head: &spgwpb.Head{
			CommandId: spgwpb.CmdID_CmdDeliverAck,
			SeqId:     v.GetSequenceID(),
			RecvTime:  time.Now().UnixMilli(),
		},
		RawMsg: data,
		DeliverAck: &spgwpb.DeliverAck{
			MsgId:  v.MsgID,
			Result: uint32(v.Result),
		},
	}
}

func pduToMsg_connect20(v *cmpp20.PduConnect, data []byte) *spgwpb.Msg {
	return &spgwpb.Msg{
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
}

func pduToMsg_connectResp20(v *cmpp20.PduConnectResp, data []byte) *spgwpb.Msg {
	return &spgwpb.Msg{
		Head: &spgwpb.Head{
			CommandId: spgwpb.CmdID_CmdBindAck,
			SeqId:     v.GetSequenceID(),
			Status:    uint32(v.Status),
			RecvTime:  time.Now().UnixMilli(),
		},
		RawMsg: data,
		BindAccountRsp: &spgwpb.BindAccountRsp{
			AuthenticatorIsmg: v.AuthenticatorISMG[:],
		},
	}
}

func pduToMsg_activeTest20(pdu protocol.PDU) (msg, ack *spgwpb.Msg) {
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

func pduToMsg_activeTestResp20(pdu protocol.PDU) *spgwpb.Msg {
	return &spgwpb.Msg{
		Head: &spgwpb.Head{
			CommandId: spgwpb.CmdID_CmdActiveTestAck,
			SeqId:     pdu.GetSequenceID(),
			RecvTime:  time.Now().UnixMilli(),
		},
	}
}

func pduToMsg_terminate20(pdu protocol.PDU) (msg, ack *spgwpb.Msg) {
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
// pduToMsgCmpp20 分发器
// ---------------------------------------------------------------------------

func pduToMsgCmpp20(pdu protocol.PDU, data []byte) (*spgwpb.Msg, *spgwpb.Msg, error) {
	switch pdu.GetCommand().ToUint32() {

	case uint32(cmpp.CommandSubmit):
		return pduToMsg_submit20(pdu.(*cmpp20.PduSubmit), data), nil, nil

	case uint32(cmpp.CommandSubmitResp):
		return pduToMsg_submitResp20(pdu.(*cmpp20.PduSubmitResp), data), nil, nil

	case uint32(cmpp.CommandDeliver):
		msg, err := pduToMsg_deliver20(pdu.(*cmpp20.PduDeliver), data)
		return msg, nil, err

	case uint32(cmpp.CommandDeliverResp):
		return pduToMsg_deliverResp20(pdu.(*cmpp20.PduDeliverResp), data), nil, nil

	case uint32(cmpp.CommandConnect):
		req := pdu.(*cmpp20.PduConnect)
		if len(req.SourceAddr) == 0 || len(req.AuthenticatorSource) == 0 || req.Timestamp == 0 {
			ack := spgwpb.NewMsg(spgwpb.CmdID_CmdBindAck, req.GetSequenceID())
			ack.Head.Status = consts.ErrCmppCodeMsgStruct
			return nil, ack, fmt.Errorf("connect source addr or authenticator source or timestamp is empty")
		}
		if req.Version != cmpp.Version20 && req.Version != 0x21 {
			ack := spgwpb.NewMsg(spgwpb.CmdID_CmdBindAck, req.GetSequenceID())
			ack.Head.Status = consts.ErrCmppCodeVersionErr
			return nil, ack, fmt.Errorf("connect version %v not supported", req.Version)
		}
		if !IsValidTimestamp(req.Timestamp) {
			ack := spgwpb.NewMsg(spgwpb.CmdID_CmdBindAck, req.GetSequenceID())
			ack.Head.Status = consts.ErrCmppCodeAuthErr // 时间戳范围不对
			return nil, ack, fmt.Errorf("connect timestamp %v is out of range", req.Timestamp)
		}
		return pduToMsg_connect20(req, data), nil, nil

	case uint32(cmpp.CommandConnectResp):
		return pduToMsg_connectResp20(pdu.(*cmpp20.PduConnectResp), data), nil, nil

	case uint32(cmpp.CommandActiveTest):
		msg, ack := pduToMsg_activeTest20(pdu)
		return msg, ack, nil

	case uint32(cmpp.CommandActiveTestResp):
		return pduToMsg_activeTestResp20(pdu), nil, nil

	case uint32(cmpp.CommandTerminate):
		msg, ack := pduToMsg_terminate20(pdu)
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
// msgToPdu 辅助函数（每种 Msg 类型一个独立函数）
// ---------------------------------------------------------------------------

func msgToPdu_submitReq20(sr *spgwpb.SubmitReq, seqID uint32) *cmpp20.PduSubmit {
	return &cmpp20.PduSubmit{
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
		TpPID:              uint8(sr.TpPid),
		TpUDHI:             uint8(sr.TpUdhi),
		MsgFmt:             uint8(sr.MsgFmt),
		MsgSrc:             sr.MsgSrc,
		FeeType:            sr.FeeType,
		FeeCode:            sr.FeeCode,
		ValIDTime:          sr.ValidTime,
		AtTime:             sr.AtTime,
		SrcID:              sr.SrcId,
		DestUsrTL:          uint8(sr.DestUsrTl),
		DestTerminalID:     sr.DestTerminalId,
		MsgLength:          uint8(sr.MsgLength),
		MsgContent:         sr.MsgContent,
	}
}

func msgToPdu_submitAck20(sa *spgwpb.SubmitAck, seqID uint32) *cmpp20.PduSubmitResp {
	return &cmpp20.PduSubmitResp{
		Header: cmpp.Header{
			SequenceID: seqID,
			CommandID:  cmpp.CommandSubmitResp,
		},
		MsgID:  sa.MsgId,
		Result: uint8(sa.Result),
	}
}

func msgToPdu_deliverReq20(dr *spgwpb.DeliverReq, seqID uint32) *cmpp20.PduDeliver {
	return &cmpp20.PduDeliver{
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
		RegisteredDeliver: uint8(dr.DeliverBase.RegisteredDelivery),
		MsgLength:         uint8(dr.MsgLength),
		MsgContent:        dr.MsgContent,
		Reserved:          dr.LinkId,
	}
}

func msgToPdu_deliverAck20(msg *spgwpb.Msg, seqID uint32) *cmpp20.PduDeliverResp {
	p := &cmpp20.PduDeliverResp{
		Header: cmpp.Header{
			SequenceID: seqID,
			CommandID:  cmpp.CommandDeliverResp,
		},
	}

	if msg.DeliverAck != nil {
		p.MsgID = msg.DeliverAck.MsgId
		p.Result = uint8(msg.DeliverAck.Result)
		return p
	}
	if msg.ReportAck != nil {
		p.MsgID = msg.ReportAck.MsgId
		p.Result = uint8(msg.ReportAck.Result)
	}
	return p
}

func msgToPdu_reportReq20(rr *spgwpb.ReportReq, seqID uint32) *cmpp20.PduDeliver {
	rpt := cmpp.SubPduDeliveryContent{
		MsgID:          rr.MsgId,
		Stat:           rr.Stat,
		SubmitTime:     rr.SubmitTime,
		DoneTime:       rr.DoneTime,
		DestTerminalID: rr.DestTerminalId,
		SMSCSequence:   rr.SmscSequence,
	}
	content, _ := rpt.IEncode()
	return &cmpp20.PduDeliver{
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
		RegisteredDeliver: uint8(rr.DeliverBase.RegisteredDelivery),

		MsgLength:  uint8(len(content)),
		MsgContent: content,
	}
}

func msgToPdu_reportAck20(ra *spgwpb.ReportAck, seqID uint32) *cmpp20.PduDeliverResp {
	return &cmpp20.PduDeliverResp{
		Header: cmpp.Header{
			SequenceID: seqID,
			CommandID:  cmpp.CommandDeliverResp,
		},
		MsgID:  ra.MsgId,
		Result: uint8(ra.Result),
	}
}

func msgToPdu_bindReq20(br *spgwpb.BindAccountReq, seqID uint32) *cmpp20.PduConnect {
	v := &cmpp20.PduConnect{
		Header: cmpp.Header{
			CommandID:  cmpp.CommandConnect,
			SequenceID: seqID,
		},
		SourceAddr: br.Spid,
		// AuthenticatorSource: br.Auth[:],
		Version:   cmpp.Version20,
		Timestamp: br.Timestamp,
	}
	copy(v.AuthenticatorSource[:], br.AuthenticatorSource)
	return v
}

func msgToPdu_bindAck20(br *spgwpb.BindAccountRsp, status uint32, seqID uint32) *cmpp20.PduConnectResp {
	v2 := &cmpp20.PduConnectResp{
		Header: cmpp.Header{
			SequenceID: seqID,
			CommandID:  cmpp.CommandConnectResp,
		},
	}
	v2.Status = cmpp20.ConnectRespStatus(uint8(status))
	if v2.Status == 1 {
		zap.S().Infof("status = %d", v2.Status)
	}
	if br != nil {
		buff := bytes.NewBuffer(nil)
		binary.Write(buff, binary.BigEndian, v2.Status)
		copy(v2.AuthenticatorISMG[:], cmpp.GenConnectRespAuthISMG(buff.Bytes(), br.AuthenticatorSource, br.Password))
	}
	v2.Version = cmpp.Version20
	return v2
}

func msgToPdu_activeTestReq20(seqID uint32) *cmpp20.PduActiveTest {
	return &cmpp20.PduActiveTest{
		Header: cmpp.Header{
			CommandID:  cmpp.CommandActiveTest,
			SequenceID: seqID,
		},
	}
}

func msgToPdu_activeTestAck20(seqID uint32) *cmpp20.PduActiveTestResp {
	return &cmpp20.PduActiveTestResp{
		Header: cmpp.Header{
			CommandID:  cmpp.CommandActiveTestResp,
			SequenceID: seqID,
		},
	}
}
