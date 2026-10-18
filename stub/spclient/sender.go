package main

import (
	"context"
	"strconv"

	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

func (c *Client) RunSender(ctx context.Context, tracker *Tracker) error {
	lim := rate.NewLimiter(rate.Limit(c.cfg.Rate), c.cfg.Rate)

	phone := c.cfg.PhoneFrom
	content := c.cfg.Sign + c.cfg.Content

	for i := 0; i < c.cfg.Total; i++ {
		if err := lim.Wait(ctx); err != nil {
			return err
		}

		seqID := c.NextSeq()
		phoneStr := strconv.FormatUint(phone, 10)

		params := SubmitParams{
			SeqID:     seqID,
			MsgID:     0,
			Phone:     phoneStr,
			Content:   content,
			ServiceID: c.cfg.ServiceID,
			FeeType:   c.cfg.FeeType,
			SrcID:     c.cfg.SrcID,
			MsgSrc:    c.cfg.Spid,
		}
		data, err := c.codec.BuildSubmit(params)
		if err != nil {
			tracker.IncSendErr()
			zap.S().Errorf("build submit seq=%d phone=%s: %v", seqID, phoneStr, err)
			continue
		}

		tracker.RegisterSubmit(seqID, phoneStr)
		if err := c.writeRaw(data); err != nil {
			tracker.IncSendErr()
			zap.S().Errorf("write submit seq=%d: %v", seqID, err)
			return err
		}

		phone++
		if phone > c.cfg.PhoneTo {
			phone = c.cfg.PhoneFrom
		}

		if (i+1)%1000 == 0 {
			zap.S().Infof("sent %d / %d", i+1, c.cfg.Total)
		}
	}
	return nil
}
