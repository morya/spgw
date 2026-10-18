package link

import "errors"

var (
	// ErrQueueFull 链路发送队列已满，消息未被接受。调用方据此触发背压策略。
	ErrQueueFull = errors.New("link queue full")

	// ErrLinkClosed 链路已关闭或正在关闭，不可恢复，不应重试同一链路。
	ErrLinkClosed = errors.New("link closed")
)
