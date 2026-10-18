package main

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	protocol "github.com/hujm2023/go-sms-protocol"
	"github.com/hujm2023/go-sms-protocol/cmpp"
	"github.com/hujm2023/go-sms-protocol/cmpp/cmpp20"
	"github.com/hujm2023/go-sms-protocol/cmpp/cmpp30"
	"go.uber.org/zap"
)

type Client struct {
	cfg   *Config
	codec Codec

	conn    net.Conn
	writeMu sync.Mutex

	seq atomic.Uint32

	Tracker *Tracker

	closeOnce sync.Once
	closed    chan struct{}
}

func DialAndBind(ctx context.Context, cfg *Config) (*Client, error) {
	codec, err := NewCodec(cfg.Protocol)
	if err != nil {
		return nil, err
	}

	d := net.Dialer{Timeout: cfg.DialTimeout}
	conn, err := d.DialContext(ctx, "tcp", cfg.Host)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", cfg.Host, err)
	}

	c := &Client{
		cfg:    cfg,
		codec:  codec,
		conn:   conn,
		closed: make(chan struct{}),
	}
	nn := time.Now().Unix()
	c.seq.Store(uint32(nn))

	if err := c.bind(); err != nil {
		_ = conn.Close()
		return nil, err
	}
	zap.S().Infof("bind ok: host=%s protocol=%s spid=%s", cfg.Host, cfg.Protocol, cfg.Spid)
	return c, nil
}

func (c *Client) NextSeq() uint32 { return c.seq.Add(1) }

func (c *Client) bind() error {
	seqID := c.NextSeq()
	data, err := c.codec.BuildConnect(seqID, c.cfg.Spid, c.cfg.Password)
	if err != nil {
		return fmt.Errorf("encode connect: %w", err)
	}
	if err := c.writeRaw(data); err != nil {
		return fmt.Errorf("send connect: %w", err)
	}

	_ = c.conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	defer c.conn.SetReadDeadline(time.Time{})

	pdu, err := c.readOne()
	if err != nil {
		return fmt.Errorf("read connect_resp: %w", err)
	}

	switch r := pdu.(type) {
	case *cmpp30.ConnectResp:
		if r.Status != 0 {
			return fmt.Errorf("connect refused: status=%d", r.Status)
		}
	case *cmpp20.PduConnectResp:
		if r.Status != 0 {
			return fmt.Errorf("connect refused: status=%d", r.Status)
		}
	default:
		return fmt.Errorf("unexpected pdu in bind: %T", pdu)
	}
	return nil
}

func (c *Client) writeRaw(data []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	_, err := c.conn.Write(data)
	return err
}

func (c *Client) readOne() (protocol.PDU, error) {
	var hdr [4]byte
	if _, err := io.ReadFull(c.conn, hdr[:]); err != nil {
		return nil, err
	}
	total := binary.BigEndian.Uint32(hdr[:])
	if total < cmpp.MinCMPPPduLength || total > 1<<20 {
		return nil, fmt.Errorf("invalid pdu length: %d", total)
	}
	buf := make([]byte, total)
	copy(buf[:4], hdr[:])
	if _, err := io.ReadFull(c.conn, buf[4:]); err != nil {
		return nil, err
	}
	return c.codec.Decode(buf)
}

func (c *Client) ReadLoop(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			zap.S().Errorf("read loop panic: %v", r)
		}
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case <-c.closed:
			return
		default:
		}

		pdu, err := c.readOne()
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
				zap.S().Infof("read loop: conn closed")
				return
			}
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			zap.S().Warnf("read pdu: %v", err)
			if c.Tracker != nil {
				c.Tracker.IncDecodeFail()
			}
			return
		}
		c.dispatch(pdu)
	}
}

func (c *Client) dispatch(pdu protocol.PDU) {
	switch p := pdu.(type) {
	case *cmpp30.SubmitResp:
		if c.Tracker != nil {
			c.Tracker.OnSubmitResp(p.Header.SequenceID, p.MsgID, p.Result)
		}
	case *cmpp20.PduSubmitResp:
		if c.Tracker != nil {
			c.Tracker.OnSubmitResp(p.Header.SequenceID, p.MsgID, uint32(p.Result))
		}
	case *cmpp30.Deliver:
		c.handleDeliver30(p)
	case *cmpp20.PduDeliver:
		c.handleDeliver20(p)
	case *cmpp30.ActiveTest:
		data, _ := c.codec.BuildActiveTestResp(p.Header.SequenceID)
		_ = c.writeRaw(data)
	case *cmpp20.PduActiveTest:
		data, _ := c.codec.BuildActiveTestResp(p.Header.SequenceID)
		_ = c.writeRaw(data)
	case *cmpp30.Terminate:
		zap.S().Warn("peer terminated")
		c.Close()
	case *cmpp20.PduTerminate:
		zap.S().Warn("peer terminated")
		c.Close()
	default:
		zap.S().Debugf("ignore pdu: %T", pdu)
	}
}

func (c *Client) handleDeliver30(p *cmpp30.Deliver) {
	if c.Tracker != nil {
		c.Tracker.IncDeliverReq()
	}
	if p.RegisteredDeliver == 1 {
		stat, msgID := parseReportContent30(p.MsgContent)
		if c.Tracker != nil {
			c.Tracker.OnReport(msgID, stat)
		}
	}
	resp, _ := c.codec.BuildDeliverResp(p.Header.SequenceID, p.MsgID)
	_ = c.writeRaw(resp)
	if c.Tracker != nil {
		c.Tracker.IncDeliverAck()
	}
}

func (c *Client) handleDeliver20(p *cmpp20.PduDeliver) {
	if c.Tracker != nil {
		c.Tracker.IncDeliverReq()
	}
	if p.RegisteredDeliver == 1 {
		stat, msgID := parseReportContent20(p.MsgContent)
		if c.Tracker != nil {
			c.Tracker.OnReport(msgID, stat)
		}
	}
	resp, _ := c.codec.BuildDeliverResp(p.Header.SequenceID, p.MsgID)
	_ = c.writeRaw(resp)
	if c.Tracker != nil {
		c.Tracker.IncDeliverAck()
	}
}

func (c *Client) Terminate() error {
	seqID := c.NextSeq()
	data, err := c.codec.BuildTerminate(seqID)
	if err != nil {
		return err
	}
	_ = c.conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	defer c.conn.SetWriteDeadline(time.Time{})
	return c.writeRaw(data)
}

func (c *Client) Close() {
	c.closeOnce.Do(func() {
		close(c.closed)
		_ = c.conn.Close()
	})
}

func parseReportContent30(b []byte) (stat string, msgID uint64) {
	if len(b) < 8 {
		return "", 0
	}
	msgID = binary.BigEndian.Uint64(b[:8])
	if len(b) >= 8+7 {
		stat = string(b[8 : 8+7])
	}
	return
}

func parseReportContent20(b []byte) (stat string, msgID uint64) {
	if len(b) < 8 {
		return "", 0
	}
	msgID = binary.BigEndian.Uint64(b[:8])
	if len(b) >= 8+7 {
		stat = string(b[8 : 8+7])
	}
	return
}

// ---------------------------------------------------------------------------
// 测试辅助函数
// ---------------------------------------------------------------------------

// DialOnly 只建立 TCP 连接，不执行 bind
func DialOnly(ctx context.Context, host string, timeout time.Duration) (net.Conn, error) {
	d := net.Dialer{Timeout: timeout}
	return d.DialContext(ctx, "tcp", host)
}

// SendRaw 发送原始字节到连接
func SendRaw(conn net.Conn, data []byte, timeout time.Duration) error {
	if timeout > 0 {
		_ = conn.SetWriteDeadline(time.Now().Add(timeout))
		defer conn.SetWriteDeadline(time.Time{})
	}
	_, err := conn.Write(data)
	return err
}

// ReadRawResp 从连接读取一个 PDU 响应（先试 cmpp30 解码，失败再试 cmpp20）
func ReadRawResp(conn net.Conn, codec Codec, timeout time.Duration) (protocol.PDU, error) {
	if timeout > 0 {
		_ = conn.SetReadDeadline(time.Now().Add(timeout))
		defer conn.SetReadDeadline(time.Time{})
	}
	var hdr [4]byte
	if _, err := io.ReadFull(conn, hdr[:]); err != nil {
		return nil, err
	}
	total := binary.BigEndian.Uint32(hdr[:])
	if total < cmpp.MinCMPPPduLength || total > 1<<20 {
		return nil, fmt.Errorf("invalid pdu length: %d", total)
	}
	buf := make([]byte, total)
	copy(buf[:4], hdr[:])
	if _, err := io.ReadFull(conn, buf[4:]); err != nil {
		return nil, err
	}
	return codec.Decode(buf)
}

// SendSubmitAndWait 发送 submit 并同步等待 submit_resp（跳过 ActiveTest 等无关 PDU）
func (c *Client) SendSubmitAndWait(params SubmitParams, timeout time.Duration) (protocol.PDU, error) {
	data, err := c.codec.BuildSubmit(params)
	if err != nil {
		return nil, fmt.Errorf("build submit: %w", err)
	}
	if c.Tracker != nil {
		c.Tracker.RegisterSubmit(params.SeqID, params.Phone)
	}
	if err := c.writeRaw(data); err != nil {
		return nil, fmt.Errorf("write submit: %w", err)
	}
	if timeout > 0 {
		_ = c.conn.SetReadDeadline(time.Now().Add(timeout))
		defer c.conn.SetReadDeadline(time.Time{})
	}
	for {
		pdu, err := c.readOne()
		if err != nil {
			return nil, err
		}
		switch p := pdu.(type) {
		case *cmpp30.SubmitResp:
			return p, nil
		case *cmpp20.PduSubmitResp:
			return p, nil
		case *cmpp30.ActiveTest:
			d, _ := c.codec.BuildActiveTestResp(p.Header.SequenceID)
			_ = c.writeRaw(d)
		case *cmpp20.PduActiveTest:
			d, _ := c.codec.BuildActiveTestResp(p.Header.SequenceID)
			_ = c.writeRaw(d)
		}
	}
}

// SendSubmitExAndWait 发送扩展 submit（群发/长消息）并同步等待 submit_resp
func (c *Client) SendSubmitExAndWait(params SubmitExParams, timeout time.Duration) (protocol.PDU, error) {
	data, err := BuildSubmitEx(params)
	if err != nil {
		return nil, fmt.Errorf("build submit ex: %w", err)
	}
	if err := c.writeRaw(data); err != nil {
		return nil, fmt.Errorf("write submit: %w", err)
	}
	if timeout > 0 {
		_ = c.conn.SetReadDeadline(time.Now().Add(timeout))
		defer c.conn.SetReadDeadline(time.Time{})
	}
	for {
		pdu, err := c.readOne()
		if err != nil {
			return nil, err
		}
		switch p := pdu.(type) {
		case *cmpp30.SubmitResp:
			return p, nil
		case *cmpp20.PduSubmitResp:
			return p, nil
		case *cmpp30.ActiveTest:
			d, _ := c.codec.BuildActiveTestResp(p.Header.SequenceID)
			_ = c.writeRaw(d)
		case *cmpp20.PduActiveTest:
			d, _ := c.codec.BuildActiveTestResp(p.Header.SequenceID)
			_ = c.writeRaw(d)
		}
	}
}
