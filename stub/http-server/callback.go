package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

type ReportItem struct {
	Status       string `json:"status"`
	Phone        string `json:"phone"`
	FeeCount     int    `json:"feeCount"`
	DeliveryMsg  string `json:"deliveryMsg"`
	DeliveryCode string `json:"deliveryCode"`
	DeliveryTime string `json:"deliveryTime"`
	ExtendCode   string `json:"extendCode"`
	MsgID        string `json:"msgId"`
	SpMsgID      string `json:"spMsgId"`
	AppID        string `json:"appId"`
}

type Callback struct {
	cfg    *Config
	in     chan SentMessage
	wg     sync.WaitGroup
	closed chan struct{}
	rng    *rand.Rand
	rngMu  sync.Mutex
	client *http.Client
}

func NewCallback(cfg *Config) *Callback {
	return &Callback{
		cfg:    cfg,
		in:     make(chan SentMessage, 10000),
		closed: make(chan struct{}),
		rng:    rand.New(rand.NewSource(time.Now().UnixNano())),
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (cb *Callback) Enqueue(m SentMessage) {
	m.SentAt = time.Now()
	select {
	case cb.in <- m:
	default:
		zap.S().Warnf("callback queue full, drop msgId=%s", m.MsgID)
	}
}

func (cb *Callback) Start(ctx context.Context) {
	cb.wg.Add(1)
	go cb.loop(ctx)
}

func (cb *Callback) Drain(timeout time.Duration) {
	close(cb.closed)
	done := make(chan struct{})
	go func() {
		cb.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(timeout):
		zap.S().Warn("callback drain timeout")
	}
}

func (cb *Callback) loop(ctx context.Context) {
	defer cb.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			zap.S().Errorf("callback loop panic: %v", r)
		}
	}()

	pending := make([]ReportItem, 0, cb.cfg.CallbackBatchSize)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	flush := func(force bool) {
		if len(pending) == 0 {
			return
		}
		if !force && len(pending) < cb.cfg.CallbackBatchSize {
			return
		}
		cb.deliver(pending)
		pending = pending[:0]
	}

	for {
		select {
		case <-ctx.Done():
			flush(true)
			return
		case <-cb.closed:
			for {
				select {
				case m := <-cb.in:
					pending = append(pending, cb.buildReport(m))
				default:
					flush(true)
					return
				}
			}
		case m := <-cb.in:
			if d := cb.cfg.CallbackDelay - time.Since(m.SentAt); d > 0 {
				time.Sleep(d)
			}
			pending = append(pending, cb.buildReport(m))
			flush(false)
		case <-ticker.C:
			flush(true)
		}
	}
}

func (cb *Callback) buildReport(m SentMessage) ReportItem {
	cb.rngMu.Lock()
	r := cb.rng.Float64()
	cb.rngMu.Unlock()

	now := time.Now().Format("2006-01-02 15:04:05")
	if r < cb.cfg.CallbackFailRate {
		return ReportItem{
			Status:       "UNDELIV",
			Phone:        m.Phone,
			FeeCount:     0,
			DeliveryMsg:  "DELIVERY FAILED",
			DeliveryCode: "DELIVRD:001",
			DeliveryTime: now,
			ExtendCode:   m.ExtendNo,
			MsgID:        m.MsgID,
			SpMsgID:      m.SpMsgID,
			AppID:        m.AppID,
		}
	}
	return ReportItem{
		Status:       "DELIVRD",
		Phone:        m.Phone,
		FeeCount:     1,
		DeliveryMsg:  "DELIVRD",
		DeliveryCode: "",
		DeliveryTime: now,
		ExtendCode:   m.ExtendNo,
		MsgID:        m.MsgID,
		SpMsgID:      m.SpMsgID,
		AppID:        m.AppID,
	}
}

func (cb *Callback) deliver(items []ReportItem) {
	if cb.cfg.CallbackURL == "" {
		zap.S().Infow("report (log only)", "count", len(items), "first_msgId", items[0].MsgID)
		return
	}
	body, err := json.Marshal(items)
	if err != nil {
		zap.S().Errorf("marshal report: %v", err)
		return
	}

	var lastErr error
	for i := 0; i <= cb.cfg.CallbackRetry; i++ {
		if i > 0 {
			time.Sleep(cb.cfg.CallbackRetryInterval)
		}
		err := cb.doPost(body)
		if err == nil {
			zap.S().Infow("report sent", "count", len(items), "attempt", i+1)
			return
		}
		lastErr = err
		zap.S().Warnf("report post attempt=%d err=%v", i+1, err)
	}
	zap.S().Errorf("report failed after retries: %v", lastErr)
}

func (cb *Callback) doPost(body []byte) error {
	req, err := http.NewRequest(http.MethodPost, cb.cfg.CallbackURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json;charset=utf-8")

	resp, err := cb.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http status %d body=%s", resp.StatusCode, string(respBody))
	}
	var ack struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(respBody, &ack); err != nil {
		return fmt.Errorf("unmarshal ack: %w body=%s", err, string(respBody))
	}
	if ack.Status != "0" {
		return fmt.Errorf("ack status=%s body=%s", ack.Status, string(respBody))
	}
	return nil
}
