package microx

import (
	"bytes"
	"context"

	"go-micro.dev/v5/metadata"
	"go-micro.dev/v5/server"
	"go.uber.org/zap"

	"github.com/morya/gommon/mytrace"
)

func LogHandle(h server.HandlerFunc) server.HandlerFunc {
	return func(ctx context.Context, req server.Request, rsp any) error {
		err := h(ctx, req, rsp)
		funcName := mytrace.GetCallerName()

		l := zap.S().With(zap.Any("req", req.Body()), zap.Any("ack", rsp), zap.Any("func", funcName))

		if err != nil {
			l = l.With(zap.Error(err))
			md, _ := metadata.FromContext(ctx)
			if md != nil {
				l = l.With(zap.Any("meta", md))
			}
			l.Error("serve rpc failed")
		} else {
			l.Debugf("serve rpc success")
		}
		return err
	}
}

func LogSubscribe(s server.SubscriberFunc) server.SubscriberFunc {
	return func(ctx context.Context, msg server.Message) error {
		body := string(bytes.TrimSpace(msg.Body()))
		zap.S().With(zap.Any("topic", msg.Topic()), zap.Any("body", body)).Debugf("recv msg")

		return s(ctx, msg)
	}
}
