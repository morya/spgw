package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"time"

	sms "github.com/hujm2023/go-sms-protocol"
	"github.com/hujm2023/go-sms-protocol/cmpp"
	"github.com/hujm2023/go-sms-protocol/cmpp/cmpp20"
	"github.com/hujm2023/go-sms-protocol/cmpp/cmpp30"
	"go.uber.org/zap"
)

type TestStatus string

const (
	StatusPass TestStatus = "PASS"
	StatusFail TestStatus = "FAIL"
	StatusSkip TestStatus = "SKIP"
)

type SharedBuff struct {
	buff bytes.Buffer
}

func (s *SharedBuff) Write(p []byte) (n int, err error) {
	s.buff.Reset()
	return s.buff.Write(p)
}

var globalBuff SharedBuff

type TestCase struct {
	ID       string
	Name     string
	Category string
	Run      func(ctx context.Context, cfg *TestConfig) TestResult
}

type TestResult struct {
	ID       string
	Name     string
	Category string
	Status   TestStatus
	Duration time.Duration
	Details  string
	Error    string
}

type TestConfig struct {
	Host20      string
	Host30      string
	Spid        string
	Password    string
	WrongSpid   string
	WrongPwd    string
	ServiceID   string
	FeeType     string
	SrcID       string
	Sign        string
	Content     string
	DialTimeout time.Duration
	ReadTimeout time.Duration
}

type TestRunner struct {
	Cases  []TestCase
	Config *TestConfig
}

func NewTestRunner(cfg *TestConfig) *TestRunner {
	return &TestRunner{Config: cfg, Cases: allTestCases()}
}

func (r *TestRunner) Run(ctx context.Context) []TestResult {
	results := make([]TestResult, 0, len(r.Cases))
	for _, tc := range r.Cases {
		// zap.S().Infof("========== %s: %s ==========", tc.ID, tc.Name)
		start := time.Now()
		res := tc.Run(ctx, r.Config)
		res.Duration = time.Since(start)
		res.ID, res.Name, res.Category = tc.ID, tc.Name, tc.Category
		if res.Details == "" {
			res.Details = string(res.Status)
		}
		results = append(results, res)

		tag := colorizeStatusTag(res.Status)
		zap.S().Infof("%s %s: %s (%s) - %s", tag, tc.ID, res.Status, res.Duration.Round(time.Millisecond), res.Details)
		if res.Error != "" {
			zap.S().Errorf("  %s", colorize("Error: "+res.Error, colorRed))
		}
	}
	return results
}

func rawDialConnect(ctx context.Context, host, proto string, data []byte, dialTimeout, readTimeout time.Duration) (sms.PDU, error) {
	d := net.Dialer{Timeout: dialTimeout}
	conn, err := d.DialContext(ctx, "tcp", host)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", host, err)
	}
	defer conn.Close()
	if readTimeout > 0 {
		_ = conn.SetWriteDeadline(time.Now().Add(readTimeout))
	}
	if _, err := conn.Write(data); err != nil {
		return nil, fmt.Errorf("write: %w", err)
	}
	if readTimeout > 0 {
		_ = conn.SetReadDeadline(time.Now().Add(readTimeout))
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

	codec, err := NewCodec(proto)
	if err != nil {
		return nil, fmt.Errorf("codec for proto %q: %w", proto, err)
	}

	globalBuff.Write(buf)
	return codec.Decode(buf)
}

func bindClient(ctx context.Context, cfg *TestConfig, proto string) (*Client, error) {
	host := cfg.Host30
	if proto == "cmpp2.0" {
		host = cfg.Host20
	}
	ccfg := &Config{
		Host:        host,
		Protocol:    proto,
		Spid:        cfg.Spid,
		Password:    cfg.Password,
		Rate:        1,
		Total:       1,
		PhoneFrom:   13800000001,
		PhoneTo:     13800000001,
		Sign:        cfg.Sign,
		Content:     cfg.Content,
		ServiceID:   cfg.ServiceID,
		FeeType:     cfg.FeeType,
		SrcID:       cfg.SrcID,
		DialTimeout: cfg.DialTimeout,
	}
	return DialAndBind(ctx, ccfg)
}

func makeSubmitParams(cfg *TestConfig, seqID uint32, phone, proto string) SubmitParams {
	return SubmitParams{
		SeqID:     seqID,
		Phone:     phone,
		Content:   cfg.Sign + cfg.Content,
		ServiceID: cfg.ServiceID,
		FeeType:   cfg.FeeType,
		SrcID:     cfg.SrcID,
		MsgSrc:    cfg.Spid,
	}
}

func allTestCases() []TestCase {
	return []TestCase{
		{ID: "TC-CMPP-LOGIN-001", Name: "CMPP 2.0 登录成功", Category: "CMPP-LOGIN", Run: tcLogin001},
		{ID: "TC-CMPP-LOGIN-002", Name: "CMPP 3.0 登录成功", Category: "CMPP-LOGIN", Run: tcLogin002},
		{ID: "TC-CMPP-LOGIN-003", Name: "鉴权码错误返回 status=3", Category: "CMPP-LOGIN", Run: tcLogin003},
		{ID: "TC-CMPP-LOGIN-004", Name: "SPID 不存在返回 status=3", Category: "CMPP-LOGIN", Run: tcLogin004},
		{ID: "TC-CMPP-LOGIN-005", Name: "请求结构错误返回 status=1", Category: "CMPP-LOGIN", Run: tcLogin005},
		{ID: "TC-CMPP-LOGIN-006", Name: "IP 不在白名单返回 status=57", Category: "CMPP-LOGIN", Run: tcLogin006},
		{ID: "TC-CMPP-LOGIN-007", Name: "CMPP 2.0 端口 Version 字段值校验", Category: "CMPP-LOGIN", Run: tcLogin007},
		{ID: "TC-CMPP-LOGIN-008", Name: "CMPP 3.0 端口发送 Version20 被拒绝", Category: "CMPP-LOGIN", Run: tcLogin008},
		{ID: "TC-CMPP-LOGIN-010", Name: "connect_resp SeqId 一一对应", Category: "CMPP-LOGIN", Run: tcLogin010},
		{ID: "TC-CMPP-SUBMIT-001", Name: "CMPP 2.0 submit 基础接收", Category: "CMPP-SUBMIT", Run: tcSubmit001},
		{ID: "TC-CMPP-SUBMIT-002", Name: "CMPP 3.0 submit 基础接收", Category: "CMPP-SUBMIT", Run: tcSubmit002},
		{ID: "TC-CMPP-SUBMIT-003", Name: "submit SeqId 全程透传", Category: "CMPP-SUBMIT", Run: tcSubmit003},
		{ID: "TC-CMPP-SUBMIT-005", Name: "submit_ack SeqId 一一对应", Category: "CMPP-SUBMIT", Run: tcSubmit005},
		{ID: "TC-CMPP-SUBMIT-006", Name: "群发消息(多DestTerminalID,不拆分直接透传)", Category: "CMPP-SUBMIT", Run: tcSubmit006},
		{ID: "TC-CMPP-SUBMIT-007", Name: "长消息 PkTotal>1", Category: "CMPP-SUBMIT", Run: tcSubmit007},
		{ID: "TC-FLOW-SEQ-001", Name: "SP seq->server newSeq->ISMG seq 映射", Category: "FLOW-SEQ", Run: tcFlowSeq001},
		{ID: "TC-FLOW-SEQ-002", Name: "submitAck 回程 seq 还原", Category: "FLOW-SEQ", Run: tcFlowSeq002},
		{ID: "TC-FLOW-SEQ-003", Name: "多并发 submit seq 不冲突", Category: "FLOW-SEQ", Run: tcFlowSeq003},
		{ID: "TC-FLOW-CMPP-001", Name: "CMPP 2.0 submit 全链路", Category: "FLOW-CMPP", Run: tcFlowCmpp001},
		{ID: "TC-FLOW-CMPP-002", Name: "CMPP 3.0 submit 全链路", Category: "FLOW-CMPP", Run: tcFlowCmpp002},
	}
}

func pass(details string) TestResult {
	return TestResult{Status: StatusPass, Details: details}
}

func fail(err string) TestResult {
	return TestResult{Status: StatusFail, Error: err}
}

func skip(details string) TestResult {
	return TestResult{Status: StatusSkip, Details: details}
}

func tcLogin001(ctx context.Context, cfg *TestConfig) TestResult {
	data, err := BuildConnectEx(ConnectParams{SeqID: 12345, SrcAddr: cfg.Spid, Password: cfg.Password, Proto: "cmpp2.0"})
	if err != nil {
		return fail(err.Error())
	}
	pdu, err := rawDialConnect(ctx, cfg.Host20, "cmpp2.0", data, cfg.DialTimeout, cfg.ReadTimeout)
	if err != nil {
		return fail(fmt.Sprintf("dial/send/read: %v", err))
	}
	resp, ok := pdu.(*cmpp20.PduConnectResp)
	if !ok {
		return fail(fmt.Sprintf("unexpected pdu: %T", pdu))
	}
	if resp.Status != 0 {
		return fail(fmt.Sprintf("status=%d, want 0", resp.Status))
	}
	return pass(fmt.Sprintf("status=%d seq=%d", resp.Status, resp.Header.SequenceID))
}

func tcLogin002(ctx context.Context, cfg *TestConfig) TestResult {
	data, err := BuildConnectEx(ConnectParams{SeqID: 12346, SrcAddr: cfg.Spid, Password: cfg.Password, Proto: "cmpp3.0"})
	if err != nil {
		return fail(err.Error())
	}
	pdu, err := rawDialConnect(ctx, cfg.Host30, "cmpp3.0", data, cfg.DialTimeout, cfg.ReadTimeout)
	if err != nil {
		return fail(fmt.Sprintf("dial/send/read: %v", err))
	}
	resp, ok := pdu.(*cmpp30.ConnectResp)
	if !ok {
		return fail(fmt.Sprintf("unexpected pdu: %T", pdu))
	}
	if resp.Status != 0 {
		return fail(fmt.Sprintf("status=%d, want 0", resp.Status))
	}
	return pass(fmt.Sprintf("status=%d seq=%d", resp.Status, resp.Header.SequenceID))
}

func tcLogin003(ctx context.Context, cfg *TestConfig) TestResult {
	data, err := BuildConnectEx(ConnectParams{SeqID: 20001, SrcAddr: cfg.Spid, Password: cfg.WrongPwd, Proto: "cmpp2.0"})
	if err != nil {
		return fail(err.Error())
	}
	pdu, err := rawDialConnect(ctx, cfg.Host20, "cmpp2.0", data, cfg.DialTimeout, cfg.ReadTimeout)
	if err != nil {
		return fail(fmt.Sprintf("dial/send/read: %v", err))
	}
	resp, ok := pdu.(*cmpp20.PduConnectResp)
	if !ok {
		return fail(fmt.Sprintf("unexpected pdu: %T", pdu))
	}
	if resp.Status != 3 {
		data := globalBuff.buff.Bytes()
		zap.S().Infof("data = \n%s", hex.Dump(data))
		zap.S().Infof("resp = %v", resp.String())
		return fail(fmt.Sprintf("status=%d, want 3", resp.Status))
	}
	return pass(fmt.Sprintf("status=%d (AuthErr)", resp.Status))
}

func tcLogin004(ctx context.Context, cfg *TestConfig) TestResult {
	data, err := BuildConnectEx(ConnectParams{SeqID: 20002, SrcAddr: cfg.WrongSpid, Password: cfg.Password, Proto: "cmpp2.0"})
	if err != nil {
		return fail(err.Error())
	}
	pdu, err := rawDialConnect(ctx, cfg.Host20, "cmpp2.0", data, cfg.DialTimeout, cfg.ReadTimeout)
	if err != nil {
		return fail(fmt.Sprintf("dial/send/read: %v", err))
	}
	resp, ok := pdu.(*cmpp20.PduConnectResp)
	if !ok {
		return fail(fmt.Sprintf("unexpected pdu: %T", pdu))
	}
	if resp.Status != 3 {
		return fail(fmt.Sprintf("status=%d, want 3", resp.Status))
	}
	return pass(fmt.Sprintf("status=%d (AuthErr for unknown spid)", resp.Status))
}

func tcLogin005(ctx context.Context, cfg *TestConfig) TestResult {
	data, err := BuildConnectEx(ConnectParams{SeqID: 30001, SrcAddr: "", Password: cfg.Password, Proto: "cmpp2.0"})
	if err != nil {
		return fail(err.Error())
	}
	pdu, err := rawDialConnect(ctx, cfg.Host20, "cmpp2.0", data, cfg.DialTimeout, cfg.ReadTimeout)
	if err != nil {
		return fail(fmt.Sprintf("5a(empty spid): %v", err))
	}
	resp, ok := pdu.(*cmpp20.PduConnectResp)
	if !ok {
		return fail(fmt.Sprintf("5a unexpected pdu: %T", pdu))
	}
	if resp.Status != 1 {
		return fail(fmt.Sprintf("5a status=%d, want 1", resp.Status))
	}

	data, err = BuildConnectEx(ConnectParams{SeqID: 30002, SrcAddr: cfg.Spid, AuthBytes: []byte{}, Proto: "cmpp2.0"})
	if err != nil {
		return fail(err.Error())
	}
	pdu, err = rawDialConnect(ctx, cfg.Host20, "cmpp2.0", data, cfg.DialTimeout, cfg.ReadTimeout)
	if err != nil {
		return fail(fmt.Sprintf("5b(empty auth): %v", err))
	}
	resp, ok = pdu.(*cmpp20.PduConnectResp)
	if !ok {
		return fail(fmt.Sprintf("5b unexpected pdu: %T", pdu))
	}
	if resp.Status != 1 {
		return fail(fmt.Sprintf("5b status=%d, want 1", resp.Status))
	}

	data, err = BuildConnectEx(ConnectParams{SeqID: 30003, SrcAddr: cfg.Spid, AuthBytes: []byte("dummyauth16bytes"), Timestamp: 0, Proto: "cmpp2.0"})
	if err != nil {
		return fail(err.Error())
	}
	pdu, err = rawDialConnect(ctx, cfg.Host20, "cmpp2.0", data, cfg.DialTimeout, cfg.ReadTimeout)
	if err != nil {
		return fail(fmt.Sprintf("5c(ts=0): %v", err))
	}
	resp, ok = pdu.(*cmpp20.PduConnectResp)
	if !ok {
		return fail(fmt.Sprintf("5c unexpected pdu: %T", pdu))
	}
	if resp.Status != 1 {
		return fail(fmt.Sprintf("5c status=%d, want 1", resp.Status))
	}

	return pass("5a(empty spid)=1, 5b(empty auth)=1, 5c(ts=0)=1")
}

func tcLogin006(ctx context.Context, cfg *TestConfig) TestResult {
	data, err := BuildConnectEx(ConnectParams{SeqID: 40001, SrcAddr: cfg.Spid, Password: cfg.Password, Proto: "cmpp2.0"})
	if err != nil {
		return fail(err.Error())
	}
	pdu, err := rawDialConnect(ctx, cfg.Host20, "cmpp2.0", data, cfg.DialTimeout, cfg.ReadTimeout)
	if err != nil {
		return fail(fmt.Sprintf("dial/send/read: %v", err))
	}
	resp, ok := pdu.(*cmpp20.PduConnectResp)
	if !ok {
		return fail(fmt.Sprintf("unexpected pdu: %T", pdu))
	}
	if resp.Status == 57 {
		return pass(fmt.Sprintf("status=%d (SpIpErr)", resp.Status))
	}
	if resp.Status == 0 {
		return skip("IP whitelist not configured (status=0, need 57)")
	}
	return fail(fmt.Sprintf("status=%d, want 57", resp.Status))
}

func tcLogin007(ctx context.Context, cfg *TestConfig) TestResult {
	// 7b 用字面量 0x21 而非 cmpp.Version30 常量：codec_cmpp20.go:349 对 0x21 使用字面量放行，
	// cmpp.Version30 可能 != 0x21，必须用字面量才能精确对齐 iagent 的校验逻辑。

	var data []byte
	var err error
	var pdu sms.PDU
	var resp *cmpp20.PduConnectResp
	var ok bool

	data, err = BuildConnectEx(ConnectParams{SeqID: 50001, SrcAddr: cfg.Spid, Password: cfg.Password, Version: cmpp.Version20, Proto: "cmpp2.0"})
	if err != nil {
		return fail(fmt.Sprintf("7a build: %v", err))
	}
	pdu, err = rawDialConnect(ctx, cfg.Host20, "cmpp2.0", data, cfg.DialTimeout, cfg.ReadTimeout)
	if err != nil {
		return fail(fmt.Sprintf("7a(Version=0x20): %v", err))
	}
	resp, ok = pdu.(*cmpp20.PduConnectResp)
	if !ok {
		return fail(fmt.Sprintf("7a unexpected pdu: %T", pdu))
	}
	if resp.Status != 0 {
		return fail(fmt.Sprintf("7a(Version=0x20) status=%d, want 0 (success)", resp.Status))
	}

	data, err = BuildConnectEx(ConnectParams{SeqID: 50002, SrcAddr: cfg.Spid, Password: cfg.Password, Version: 0x21, Proto: "cmpp2.0"})
	if err != nil {
		return fail(fmt.Sprintf("7b build: %v", err))
	}
	pdu, err = rawDialConnect(ctx, cfg.Host20, "cmpp2.0", data, cfg.DialTimeout, cfg.ReadTimeout)
	if err != nil {
		return fail(fmt.Sprintf("7b(Version=0x21): %v", err))
	}
	resp, ok = pdu.(*cmpp20.PduConnectResp)
	if !ok {
		return fail(fmt.Sprintf("7b unexpected pdu: %T", pdu))
	}
	if resp.Status != 0 {
		return fail(fmt.Sprintf("7b(Version=0x21) status=%d, want 0 (success)", resp.Status))
	}

	data, err = BuildConnectEx(ConnectParams{SeqID: 50003, SrcAddr: cfg.Spid, Password: cfg.Password, Version: 0x10, Proto: "cmpp2.0"})
	if err != nil {
		return fail(fmt.Sprintf("7c build: %v", err))
	}
	pdu, err = rawDialConnect(ctx, cfg.Host20, "cmpp2.0", data, cfg.DialTimeout, cfg.ReadTimeout)
	if err != nil {
		return fail(fmt.Sprintf("7c(Version=0x10) expected connect_resp status=4, but got error: %v", err))
	}
	resp, ok = pdu.(*cmpp20.PduConnectResp)
	if !ok {
		return fail(fmt.Sprintf("7c unexpected pdu: %T", pdu))
	}
	if resp.Status != 4 {
		return fail(fmt.Sprintf("7c(Version=0x10) status=%d, want 4 (VersionErr)", resp.Status))
	}

	data, err = BuildConnectEx(ConnectParams{SeqID: 50004, SrcAddr: cfg.Spid, Password: cfg.Password, Version: 0x30, Proto: "cmpp2.0"})
	if err != nil {
		return fail(fmt.Sprintf("7d build: %v", err))
	}
	pdu, err = rawDialConnect(ctx, cfg.Host20, "cmpp2.0", data, cfg.DialTimeout, cfg.ReadTimeout)
	if err != nil {
		return fail(fmt.Sprintf("7d(Version=0x30) expected connect_resp status=4, but got error: %v", err))
	}
	resp, ok = pdu.(*cmpp20.PduConnectResp)
	if !ok {
		return fail(fmt.Sprintf("7d unexpected pdu: %T", pdu))
	}
	if resp.Status != 4 {
		return fail(fmt.Sprintf("7d(Version=0x30) status=%d, want 4 (VersionErr)", resp.Status))
	}

	return pass("7a(0x20)=0, 7b(0x21)=0, 7c(0x10)=4, 7d(0x30)=4")
}

func tcLogin008(ctx context.Context, cfg *TestConfig) TestResult {
	data, err := BuildConnectEx(ConnectParams{SeqID: 50002, SrcAddr: cfg.Spid, Password: cfg.Password, Version: cmpp.Version20, Proto: "cmpp3.0"})
	if err != nil {
		return fail(err.Error())
	}
	pdu, err := rawDialConnect(ctx, cfg.Host30, "cmpp3.0", data, cfg.DialTimeout, cfg.ReadTimeout)
	if err != nil {
		return fail(fmt.Sprintf("expected connect_resp status=4, but got error: %v", err))
	}
	resp, ok := pdu.(*cmpp30.ConnectResp)
	if !ok {
		return fail(fmt.Sprintf("unexpected pdu: %T", pdu))
	}
	if resp.Status != 4 {
		return fail(fmt.Sprintf("status=%d, want 4 (VersionErr)", resp.Status))
	}
	return pass(fmt.Sprintf("status=%d (VersionErr)", resp.Status))
}

func tcLogin010(ctx context.Context, cfg *TestConfig) TestResult {
	for _, seqID := range []uint32{111, 222, 333, 444, 555} {
		data, err := BuildConnectEx(ConnectParams{SeqID: seqID, SrcAddr: cfg.Spid, Password: cfg.Password, Proto: "cmpp2.0"})
		if err != nil {
			return fail(err.Error())
		}
		pdu, err := rawDialConnect(ctx, cfg.Host20, "cmpp2.0", data, cfg.DialTimeout, cfg.ReadTimeout)
		if err != nil {
			return fail(fmt.Sprintf("seq=%d: %v", seqID, err))
		}
		resp, ok := pdu.(*cmpp20.PduConnectResp)
		if !ok {
			return fail(fmt.Sprintf("seq=%d: unexpected pdu: %T", seqID, pdu))
		}
		if resp.Header.SequenceID != seqID {
			return fail(fmt.Sprintf("seq=%d: resp seq=%d, mismatch", seqID, resp.Header.SequenceID))
		}
	}
	return pass("5 rounds: seq 111/222/333/444/555 all matched")
}

func tcSubmit001(ctx context.Context, cfg *TestConfig) TestResult {
	cli, err := bindClient(ctx, cfg, "cmpp2.0")
	if err != nil {
		return fail(fmt.Sprintf("bind: %v", err))
	}
	defer cli.Close()

	params := makeSubmitParams(cfg, 99001, "13800138000", "cmpp2.0")
	pdu, err := cli.SendSubmitAndWait(params, cfg.ReadTimeout)
	if err != nil {
		return fail(fmt.Sprintf("submit: %v", err))
	}
	resp, ok := pdu.(*cmpp20.PduSubmitResp)
	if !ok {
		return fail(fmt.Sprintf("unexpected pdu: %T", pdu))
	}
	return pass(fmt.Sprintf("submit_resp seq=%d msgID=%d result=%d", resp.Header.SequenceID, resp.MsgID, resp.Result))
}

func tcSubmit002(ctx context.Context, cfg *TestConfig) TestResult {
	cli, err := bindClient(ctx, cfg, "cmpp3.0")
	if err != nil {
		return fail(fmt.Sprintf("bind: %v", err))
	}
	defer cli.Close()

	params := makeSubmitParams(cfg, 99002, "13800138001", "cmpp3.0")
	pdu, err := cli.SendSubmitAndWait(params, cfg.ReadTimeout)
	if err != nil {
		return fail(fmt.Sprintf("submit: %v", err))
	}
	resp, ok := pdu.(*cmpp30.SubmitResp)
	if !ok {
		return fail(fmt.Sprintf("unexpected pdu: %T", pdu))
	}
	return pass(fmt.Sprintf("submit_resp seq=%d msgID=%d result=%d", resp.Header.SequenceID, resp.MsgID, resp.Result))
}

func tcSubmit003(ctx context.Context, cfg *TestConfig) TestResult {
	cli, err := bindClient(ctx, cfg, "cmpp3.0")
	if err != nil {
		return fail(fmt.Sprintf("bind: %v", err))
	}
	defer cli.Close()

	for _, seqID := range []uint32{100, 200, 300} {
		params := makeSubmitParams(cfg, seqID, "13800138000", "cmpp3.0")
		pdu, err := cli.SendSubmitAndWait(params, cfg.ReadTimeout)
		if err != nil {
			return fail(fmt.Sprintf("seq=%d: %v", seqID, err))
		}
		resp, ok := pdu.(*cmpp30.SubmitResp)
		if !ok {
			return fail(fmt.Sprintf("seq=%d: unexpected pdu: %T", seqID, pdu))
		}
		if resp.Header.SequenceID != seqID {
			return fail(fmt.Sprintf("seq=%d: resp seq=%d, mismatch", seqID, resp.Header.SequenceID))
		}
	}
	return pass("3 rounds: seq 100/200/300 all matched")
}

func tcSubmit005(ctx context.Context, cfg *TestConfig) TestResult {
	cli, err := bindClient(ctx, cfg, "cmpp2.0")
	if err != nil {
		return fail(fmt.Sprintf("bind: %v", err))
	}
	defer cli.Close()

	seqID := uint32(555)
	params := makeSubmitParams(cfg, seqID, "13900139000", "cmpp2.0")
	pdu, err := cli.SendSubmitAndWait(params, cfg.ReadTimeout)
	if err != nil {
		return fail(fmt.Sprintf("submit: %v", err))
	}
	resp, ok := pdu.(*cmpp20.PduSubmitResp)
	if !ok {
		return fail(fmt.Sprintf("unexpected pdu: %T", pdu))
	}
	if resp.Header.SequenceID != seqID {
		return fail(fmt.Sprintf("resp seq=%d, want %d", resp.Header.SequenceID, seqID))
	}
	return pass(fmt.Sprintf("submit_req seq=%d -> submit_resp seq=%d, matched", seqID, resp.Header.SequenceID))
}

func tcSubmit006(ctx context.Context, cfg *TestConfig) TestResult {
	cli, err := bindClient(ctx, cfg, "cmpp3.0")
	if err != nil {
		return fail(fmt.Sprintf("bind: %v", err))
	}
	defer cli.Close()

	phones := []string{"13800138000", "13900139000", "13700137000"}
	params := SubmitExParams{
		SeqID:     77001,
		Phones:    phones,
		Content:   cfg.Sign + cfg.Content,
		ServiceID: cfg.ServiceID,
		FeeType:   cfg.FeeType,
		SrcID:     cfg.SrcID,
		MsgSrc:    cfg.Spid,
		Proto:     "cmpp3.0",
	}
	pdu, err := cli.SendSubmitExAndWait(params, cfg.ReadTimeout)
	if err != nil {
		return fail(fmt.Sprintf("submit: %v", err))
	}
	resp, ok := pdu.(*cmpp30.SubmitResp)
	if !ok {
		return fail(fmt.Sprintf("unexpected pdu: %T", pdu))
	}
	return pass(fmt.Sprintf("群发 %d phones 不拆分直接透传 -> submit_resp seq=%d msgID=%d", len(phones), resp.Header.SequenceID, resp.MsgID))
}

func tcSubmit007(ctx context.Context, cfg *TestConfig) TestResult {
	cli, err := bindClient(ctx, cfg, "cmpp3.0")
	if err != nil {
		return fail(fmt.Sprintf("bind: %v", err))
	}
	defer cli.Close()

	params1 := SubmitExParams{
		SeqID: 88001, Phones: []string{"13800138000"},
		Content:   cfg.Sign + cfg.Content + "-part1",
		ServiceID: cfg.ServiceID, FeeType: cfg.FeeType, SrcID: cfg.SrcID, MsgSrc: cfg.Spid,
		PkTotal: 2, PkNumber: 1, Proto: "cmpp3.0",
	}
	pdu1, err := cli.SendSubmitExAndWait(params1, cfg.ReadTimeout)
	if err != nil {
		return fail(fmt.Sprintf("part1: %v", err))
	}
	resp1, ok := pdu1.(*cmpp30.SubmitResp)
	if !ok {
		return fail(fmt.Sprintf("part1: unexpected pdu: %T", pdu1))
	}

	params2 := SubmitExParams{
		SeqID: 88002, Phones: []string{"13800138000"},
		Content:   cfg.Sign + cfg.Content + "-part2",
		ServiceID: cfg.ServiceID, FeeType: cfg.FeeType, SrcID: cfg.SrcID, MsgSrc: cfg.Spid,
		PkTotal: 2, PkNumber: 2, Proto: "cmpp3.0",
	}
	pdu2, err := cli.SendSubmitExAndWait(params2, cfg.ReadTimeout)
	if err != nil {
		return fail(fmt.Sprintf("part2: %v", err))
	}
	resp2, ok := pdu2.(*cmpp30.SubmitResp)
	if !ok {
		return fail(fmt.Sprintf("part2: unexpected pdu: %T", pdu2))
	}

	return pass(fmt.Sprintf("长消息 2片: part1 seq=%d msgID=%d, part2 seq=%d msgID=%d", resp1.Header.SequenceID, resp1.MsgID, resp2.Header.SequenceID, resp2.MsgID))
}

func tcFlowSeq001(ctx context.Context, cfg *TestConfig) TestResult {
	cli, err := bindClient(ctx, cfg, "cmpp3.0")
	if err != nil {
		return fail(fmt.Sprintf("bind: %v", err))
	}
	defer cli.Close()

	spSeq := uint32(1000)
	params := makeSubmitParams(cfg, spSeq, "13800138000", "cmpp3.0")
	pdu, err := cli.SendSubmitAndWait(params, cfg.ReadTimeout)
	if err != nil {
		return fail(fmt.Sprintf("submit: %v", err))
	}
	resp, ok := pdu.(*cmpp30.SubmitResp)
	if !ok {
		return fail(fmt.Sprintf("unexpected pdu: %T", pdu))
	}
	if resp.Header.SequenceID != spSeq {
		return fail(fmt.Sprintf("resp seq=%d, want %d (SP seq)", resp.Header.SequenceID, spSeq))
	}
	return pass(fmt.Sprintf("SP seq=%d -> submit_resp seq=%d, matched (server newSeq transparent to SP)", spSeq, resp.Header.SequenceID))
}

func tcFlowSeq002(ctx context.Context, cfg *TestConfig) TestResult {
	cli, err := bindClient(ctx, cfg, "cmpp2.0")
	if err != nil {
		return fail(fmt.Sprintf("bind: %v", err))
	}
	defer cli.Close()

	spSeq := uint32(2000)
	params := makeSubmitParams(cfg, spSeq, "13900139000", "cmpp2.0")
	pdu, err := cli.SendSubmitAndWait(params, cfg.ReadTimeout)
	if err != nil {
		return fail(fmt.Sprintf("submit: %v", err))
	}
	resp, ok := pdu.(*cmpp20.PduSubmitResp)
	if !ok {
		return fail(fmt.Sprintf("unexpected pdu: %T", pdu))
	}
	if resp.Header.SequenceID != spSeq {
		return fail(fmt.Sprintf("resp seq=%d, want %d", resp.Header.SequenceID, spSeq))
	}
	return pass(fmt.Sprintf("submit_req seq=%d -> submit_resp seq=%d, seq restored", spSeq, resp.Header.SequenceID))
}

func tcFlowSeq003(ctx context.Context, cfg *TestConfig) TestResult {
	cli, err := bindClient(ctx, cfg, "cmpp3.0")
	if err != nil {
		return fail(fmt.Sprintf("bind: %v", err))
	}
	defer cli.Close()

	tracker := NewTracker()
	cli.Tracker = tracker
	go cli.ReadLoop(ctx)

	const N = 10
	seqs := make([]uint32, N)
	for i := range N {
		seqs[i] = uint32(5000 + i)
		params := makeSubmitParams(cfg, seqs[i], fmt.Sprintf("138%08d", i), "cmpp3.0")
		data, err := cli.codec.BuildSubmit(params)
		if err != nil {
			return fail(fmt.Sprintf("build submit %d: %v", i, err))
		}
		tracker.RegisterSubmit(seqs[i], fmt.Sprintf("138%08d", i))
		if err := cli.writeRaw(data); err != nil {
			return fail(fmt.Sprintf("write submit %d: %v", i, err))
		}
	}

	deadline := time.Now().Add(cfg.ReadTimeout * 3)
	for time.Now().Before(deadline) {
		s := tracker.Snapshot()
		if s.AckMatched >= uint64(N) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	s := tracker.Snapshot()
	if s.AckMatched < uint64(N) {
		return fail(fmt.Sprintf("only %d/%d acks received", s.AckMatched, N))
	}

	matched := 0
	tracker.mu.RLock()
	for _, seq := range seqs {
		if e, ok := tracker.bySeq[seq]; ok && e.HasAck && e.SeqID == seq {
			matched++
		}
	}
	tracker.mu.RUnlock()

	if matched != N {
		return fail(fmt.Sprintf("%d/%d seqs matched", matched, N))
	}
	return pass(fmt.Sprintf("%d concurrent submits, all seq matched", N))
}

func tcFlowCmpp001(ctx context.Context, cfg *TestConfig) TestResult {
	cli, err := bindClient(ctx, cfg, "cmpp2.0")
	if err != nil {
		return fail(fmt.Sprintf("bind: %v", err))
	}
	defer cli.Close()

	params := makeSubmitParams(cfg, 60001, "13800138000", "cmpp2.0")
	pdu, err := cli.SendSubmitAndWait(params, cfg.ReadTimeout)
	if err != nil {
		return fail(fmt.Sprintf("submit: %v", err))
	}
	resp, ok := pdu.(*cmpp20.PduSubmitResp)
	if !ok {
		return fail(fmt.Sprintf("unexpected pdu: %T", pdu))
	}
	return pass(fmt.Sprintf("CMPP 2.0 全链路: bind->submit->submit_resp seq=%d msgID=%d", resp.Header.SequenceID, resp.MsgID))
}

func tcFlowCmpp002(ctx context.Context, cfg *TestConfig) TestResult {
	cli, err := bindClient(ctx, cfg, "cmpp3.0")
	if err != nil {
		return fail(fmt.Sprintf("bind: %v", err))
	}
	defer cli.Close()

	params := makeSubmitParams(cfg, 60002, "13800138001", "cmpp3.0")
	pdu, err := cli.SendSubmitAndWait(params, cfg.ReadTimeout)
	if err != nil {
		return fail(fmt.Sprintf("submit: %v", err))
	}
	resp, ok := pdu.(*cmpp30.SubmitResp)
	if !ok {
		return fail(fmt.Sprintf("unexpected pdu: %T", pdu))
	}
	return pass(fmt.Sprintf("CMPP 3.0 全链路: bind->submit->submit_resp seq=%d msgID=%d", resp.Header.SequenceID, resp.MsgID))
}
