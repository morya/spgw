package main

import (
	"slices"
	"sync"
	"sync/atomic"
	"time"
)

type submitEntry struct {
	SeqID    uint32
	Phone    string
	SentAt   time.Time
	AckAt    time.Time
	AckRes   uint32
	HasAck   bool
	MsgID    uint64
	ReportAt time.Time
	HasRpt   bool
	Stat     string
}

type Tracker struct {
	mu sync.RWMutex

	bySeq   map[uint32]*submitEntry
	byMsgID map[uint64]*submitEntry

	sent     atomic.Uint64
	sendErr  atomic.Uint64
	ackTotal atomic.Uint64
	rptTotal atomic.Uint64

	// deliverReqTotal 统计所有 deliver_req（含 report），与 rptTotal（仅 report）区分
	deliverReqTotal atomic.Uint64
	deliverAckSent  atomic.Uint64

	ackUnmatched atomic.Uint64
	rptUnmatched atomic.Uint64

	decodeFail atomic.Uint64
	reconnects atomic.Uint64

	// ackResultDist 统计所有接收到的 submit_ack 的 Result 字段分布，
	// 与 ackTotal 一一对应（含 unmatched / 重复 ack），不依赖 bySeq 匹配。
	ackResultMu   sync.Mutex
	ackResultDist map[uint32]uint64
}

func NewTracker() *Tracker {
	return &Tracker{
		bySeq:         make(map[uint32]*submitEntry),
		byMsgID:       make(map[uint64]*submitEntry),
		ackResultDist: make(map[uint32]uint64),
	}
}

func (t *Tracker) RegisterSubmit(seqID uint32, phone string) {
	now := time.Now()
	t.mu.Lock()
	t.bySeq[seqID] = &submitEntry{SeqID: seqID, Phone: phone, SentAt: now}
	t.mu.Unlock()
	t.sent.Add(1)
}

func (t *Tracker) IncSendErr() { t.sendErr.Add(1) }

func (t *Tracker) IncDecodeFail() { t.decodeFail.Add(1) }

func (t *Tracker) IncReconnect() { t.reconnects.Add(1) }

func (t *Tracker) IncDeliverReq() { t.deliverReqTotal.Add(1) }

func (t *Tracker) IncDeliverAck() { t.deliverAckSent.Add(1) }

func (t *Tracker) OnSubmitResp(seqID uint32, msgID uint64, result uint32) {
	t.ackTotal.Add(1)

	// 无条件记录 result 分布，与 ackTotal 一一对应
	t.ackResultMu.Lock()
	t.ackResultDist[result]++
	t.ackResultMu.Unlock()

	now := time.Now()
	t.mu.Lock()
	defer t.mu.Unlock()
	e, ok := t.bySeq[seqID]
	if !ok {
		t.ackUnmatched.Add(1)
		return
	}
	if e.HasAck {
		return
	}
	e.HasAck = true
	e.AckAt = now
	e.AckRes = result
	e.MsgID = msgID
	if msgID != 0 {
		t.byMsgID[msgID] = e
	}
}

func (t *Tracker) OnReport(msgID uint64, stat string) {
	t.rptTotal.Add(1)
	now := time.Now()
	t.mu.Lock()
	defer t.mu.Unlock()
	e, ok := t.byMsgID[msgID]
	if !ok {
		t.rptUnmatched.Add(1)
		return
	}
	if e.HasRpt {
		return
	}
	e.HasRpt = true
	e.ReportAt = now
	e.Stat = stat
}

type Stats struct {
	Sent       uint64
	SendErr    uint64
	AckTotal   uint64
	RptTotal   uint64
	AckMatched uint64
	AckUnmatch uint64
	AckMissing uint64
	RptMatched uint64
	RptUnmatch uint64
	RptMissing uint64
	DecodeFail uint64
	Reconnects uint64

	DeliverReqTotal uint64
	DeliverAckSent  uint64

	AckLatencyP50 time.Duration
	AckLatencyP95 time.Duration
	AckLatencyP99 time.Duration

	RptLatencyP50 time.Duration
	RptLatencyP95 time.Duration
	RptLatencyP99 time.Duration

	AckResultDist map[uint32]uint64
	RptStatDist   map[string]uint64
	PrefixDist    map[string]*PrefixCount
}

type PrefixCount struct {
	Total uint64
	OK    uint64
	Err   uint64
}

func (t *Tracker) Snapshot() *Stats {
	s := &Stats{
		Sent:            t.sent.Load(),
		SendErr:         t.sendErr.Load(),
		AckTotal:        t.ackTotal.Load(),
		RptTotal:        t.rptTotal.Load(),
		AckUnmatch:      t.ackUnmatched.Load(),
		RptUnmatch:      t.rptUnmatched.Load(),
		DecodeFail:      t.decodeFail.Load(),
		Reconnects:      t.reconnects.Load(),
		DeliverReqTotal: t.deliverReqTotal.Load(),
		DeliverAckSent:  t.deliverAckSent.Load(),
		RptStatDist:     make(map[string]uint64),
		PrefixDist:      make(map[string]*PrefixCount),
	}

	t.ackResultMu.Lock()
	s.AckResultDist = make(map[uint32]uint64, len(t.ackResultDist))
	for k, v := range t.ackResultDist {
		s.AckResultDist[k] = v
	}
	t.ackResultMu.Unlock()

	t.mu.RLock()
	defer t.mu.RUnlock()

	ackLat := make([]time.Duration, 0, len(t.bySeq))
	rptLat := make([]time.Duration, 0, len(t.bySeq))

	for _, e := range t.bySeq {
		prefix := prefixOf(e.Phone, 7)
		pc, ok := s.PrefixDist[prefix]
		if !ok {
			pc = &PrefixCount{}
			s.PrefixDist[prefix] = pc
		}
		pc.Total++

		if e.HasAck {
			s.AckMatched++
			ackLat = append(ackLat, e.AckAt.Sub(e.SentAt))
			if e.AckRes == 0 {
				pc.OK++
			} else {
				pc.Err++
			}
		} else {
			s.AckMissing++
		}

		if e.HasRpt {
			s.RptMatched++
			s.RptStatDist[e.Stat]++
			rptLat = append(rptLat, e.ReportAt.Sub(e.SentAt))
		} else if e.HasAck && e.AckRes == 0 {
			s.RptMissing++
		}
	}

	s.AckLatencyP50, s.AckLatencyP95, s.AckLatencyP99 = percentile(ackLat)
	s.RptLatencyP50, s.RptLatencyP95, s.RptLatencyP99 = percentile(rptLat)
	return s
}

func prefixOf(phone string, n int) string {
	if len(phone) <= n {
		return phone
	}
	return phone[:n] + "x"
}

func percentile(xs []time.Duration) (p50, p95, p99 time.Duration) {
	if len(xs) == 0 {
		return
	}
	slices.Sort(xs)
	pick := func(p float64) time.Duration {
		idx := int(float64(len(xs)-1) * p)
		return xs[idx]
	}
	return pick(0.50), pick(0.95), pick(0.99)
}
