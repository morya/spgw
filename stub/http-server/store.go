package main

import (
	"time"

	gonanoid "github.com/matoous/go-nanoid/v2"
)

func newMsgID() string {
	// uint64 最多只能20bytes
	return gonanoid.Must(20)
}

type SentMessage struct {
	MsgID    string
	Phone    string
	SpMsgID  string
	AppID    string
	ExtendNo string
	Content  string
	SentAt   time.Time
}
