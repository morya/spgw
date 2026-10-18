package link

import (
	"context"

	"github.com/morya/spgw/spgwpb"
)

type AppMng interface {
	SendMsg(msg *spgwpb.Msg)

	GetNodeID() uint32
	GetNodeType() spgwpb.NodeType

	OnBindReq(ctx context.Context, l *Link, req *spgwpb.Msg) *spgwpb.Msg
	OnBindAck(ctx context.Context, l *Link, ack *spgwpb.Msg) error
	OnLinkBroken(l *Link) // triggered in client mode
	// OnLinkClosed 由调用方在 link.Run 返回后自行调用
}

type DummyMng struct{}

func (m *DummyMng) SendMsg(msg *spgwpb.Msg) {}

func (m *DummyMng) GetNodeID() uint32 {
	return 0
}

func (m *DummyMng) GetNodeType() spgwpb.NodeType {
	return 0
}

func (m *DummyMng) OnBindReq(ctx context.Context, l *Link, req *spgwpb.Msg) *spgwpb.Msg {
	return nil
}

func (m *DummyMng) OnBindAck(ctx context.Context, l *Link, ack *spgwpb.Msg) error { return nil }

func (m *DummyMng) OnLinkBroken(l *Link) {}

func (m *DummyMng) OnLinkClosed(ctx context.Context, l *Link) error { return nil }
