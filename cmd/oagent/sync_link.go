package oagent

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/morya/spgw/internal/db"
	"github.com/morya/spgw/internal/glog"
	"github.com/morya/spgw/internal/link"
	"github.com/morya/spgw/internal/models"
	"github.com/morya/spgw/internal/validatex"
	"github.com/morya/spgw/spgwpb"
)

// linkInfoChanged 判断 link 关键字段是否变更，决定是否需要断开重连。
// ID/Disabled 已在外层过滤，此处不比较。
func linkInfoChanged(old, cur *models.LinkInfo) bool {
	return old.RemoteAddress != cur.RemoteAddress ||
		old.LocalAddress != cur.LocalAddress ||
		old.Proto != cur.Proto ||
		old.Spid != cur.Spid ||
		old.Pswd != cur.Pswd ||
		old.OAccountID != cur.OAccountID ||
		old.Count != cur.Count
}

// syncLinksFromDB 从 DB 重新同步 LinkInfo 配置，处理 portal 后台的增/改/删。
// 不修改 mapIdToAccount（那是 CmdReloadOAccount 的职责）。
// 全程不持有 a.lock，避免与 OnLinkBroken/OnBindAck 死锁。
func (a *App) syncLinksFromDB() error {
	var arrLinks []*models.LinkInfo
	if err := db.MainDB.Model(&models.LinkInfo{}).Find(&arrLinks).Error; err != nil {
		glog.RunS().With(zap.Any("err", err)).Errorf("syncLinksFromDB: load linkinfo failed")
		return err
	}

	dbLinks := make(map[uint32]*models.LinkInfo, len(arrLinks))
	for _, item := range arrLinks {
		if item.Disabled != nil && *item.Disabled {
			continue
		}
		if err := validatex.Validate(item); err != nil {
			glog.RunS().With(zap.Any("err", err), zap.Any("link", item)).
				Errorf("syncLinksFromDB: validate failed, skip")
			continue
		}
		dbLinks[item.ID] = item
	}

	type closeAction struct {
		l         *link.Link
		reconnect bool
		newInfo   *models.LinkInfo
	}
	var toClose []closeAction

	// 扫描活跃连接 linkReady：判定删除 / 变更
	a.linkReady.Range(func(key, value any) bool {
		id, ok := key.(uint32)
		if !ok {
			return true
		}
		l, ok := value.(*link.Link)
		if !ok {
			return true
		}

		dbInfo, exists := dbLinks[id]
		if !exists {
			toClose = append(toClose, closeAction{l: l, reconnect: false})
			delete(dbLinks, id)
			return true
		}
		if linkInfoChanged(&l.Info, dbInfo) {
			toClose = append(toClose, closeAction{l: l, reconnect: true, newInfo: dbInfo})
			delete(dbLinks, id)
		}
		return true
	})

	// 扫描待连接 linkConnecting：判定删除 / 覆盖新配置（尚未建连，无需 Close）
	a.linkConnecting.Range(func(key, value any) bool {
		id, ok := key.(uint32)
		if !ok {
			return true
		}
		oldInfo, ok := value.(*models.LinkInfo)
		if !ok {
			return true
		}

		dbInfo, exists := dbLinks[id]
		if !exists {
			a.linkConnecting.Delete(id)
			delete(dbLinks, id)
			return true
		}
		if linkInfoChanged(oldInfo, dbInfo) {
			a.linkConnecting.Store(id, dbInfo)
		}
		delete(dbLinks, id)
		return true
	})

	// dbLinks 剩余的都是新增 link，加入 linkConnecting，keepReconnect 会自动建立连接
	for _, newInfo := range dbLinks {
		a.linkConnecting.Store(newInfo.ID, newInfo)
	}

	// 执行关闭。时序保证：Store(NoReconnect) 同步先于 Close()，
	// OnLinkBroken 在 Run 的 defer 中异步执行于 Close 之后，必能 LoadAndDelete 到标记。
	for _, ca := range toClose {
		a.linkNoReconnect.Store(ca.l.Info.ID, struct{}{})
		if ca.reconnect {
			a.linkConnecting.Store(ca.l.Info.ID, ca.newInfo)
		} else {
			a.linkConnecting.Delete(ca.l.Info.ID)
		}
		ca.l.Close("syncLinksFromDB")
	}

	glog.RunS().With(
		zap.Int("closed", len(toClose)),
		zap.Int("added", len(dbLinks)),
	).Infof("syncLinksFromDB done")
	return nil
}

func (a *App) onDelOLinkByID(msg *spgwpb.Msg) error {
	a.lock.Lock()
	defer a.lock.Unlock()
	req := msg.DeleteOlinkReq
	if req == nil {
		return fmt.Errorf("DeleteOlinkReq is nil")
	}

	acc, found := a.mapIdToAccount[req.AccountId]
	if !found {
		return fmt.Errorf("find account by id failed, account id:%v", req.AccountId)
	}
	l, ok := acc.RemoveLinkByID(req.LinkId)
	if !ok {
		return fmt.Errorf("find link by id failed, link id:%v", req.LinkId)
	}
	l.Close("onDelOLinkByID")

	return nil
}
