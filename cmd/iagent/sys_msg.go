package iagent

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/morya/spgw/internal/link"
	"github.com/morya/spgw/spgwpb"
)

// onDeleteILinkReq 处理 portal 广播的 DeleteILinkReq 消息。
// 根据 link_id 找到对应的 ilink 并关闭：
//  1. l.Close() -> 触发 cancel -> KeepAlive goroutine 退出并 Revoke etcd lease（/spgw/ilink-{NodeID}-{LinkID} 自动删除）；
//  2. link.Run 退出后由 handleConn 调用 OnLinkClosed 完成 mapId2Link.Delete / account.RemoveLinkByID / linkid.Release。
//
// 此处只负责触发关闭，不重复执行清理，避免与 handleConn 的 OnLinkClosed 路径产生 race。
func (a *App) onDeleteILinkReq(msg *spgwpb.Msg, logger *zap.SugaredLogger) error {
	req := msg.GetDeleteIlinkReq()
	if req == nil {
		return fmt.Errorf("delete ilink req is nil")
	}

	linkID := uint(req.GetLinkId())
	logger = logger.With(zap.Uint("linkID", linkID))

	val, ok := a.mapId2Link.Load(linkID)
	if !ok {
		logger.Infof("DeleteILinkReq: link not found, maybe already closed")
		return nil
	}

	l, ok := val.(*link.Link)
	if !ok || l == nil {
		logger.Warnf("DeleteILinkReq: link type assertion failed, value=%+v", val)
		return nil
	}

	logger = logger.With(zap.Any("linkInfo", l.Info))
	l.Close("portal DeleteILinkReq")
	logger.Infof("DeleteILinkReq: link closed")
	return nil
}
