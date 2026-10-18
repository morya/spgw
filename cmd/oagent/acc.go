package oagent

import (
	"fmt"
	"sync"

	"github.com/koss-null/list"
	"go.uber.org/zap"

	"github.com/morya/spgw/internal/link"
	"github.com/morya/spgw/internal/models"
	"github.com/morya/spgw/spgwpb"
)

type Account struct {
	models.OAccount

	lock       sync.Mutex
	linkedList list.Linked[*link.Link] // 链表，用于负载均衡分发
	mapLinks   map[uint32]*link.Link   // map，用于快速查找
}

func newAccount(oa models.OAccount) *Account {
	return &Account{
		OAccount: oa,
		mapLinks: make(map[uint32]*link.Link),
	}
}

func (p *Account) AddLink(l *link.Link) {
	p.lock.Lock()
	defer p.lock.Unlock()

	if _, ok := p.mapLinks[l.Info.ID]; ok {
		zap.S().With(zap.Any("link", l)).Errorf("link already exists")
		return
	}
	p.mapLinks[l.Info.ID] = l
	p.linkedList.PushBack(l)

	zap.S().With(
		zap.Uint32("accountID", p.ID),
		zap.String("name", p.Name),
		zap.Uint32("linkID", l.Info.ID),
		zap.Any("linkInfo", l.Info),
		zap.Int("ll-len", p.linkedList.Size()),
		zap.String("acc", fmt.Sprintf("%p", p)),
	).Infof("add link to account")
}

func (p *Account) RemoveLinkByID(id uint32) (*link.Link, bool) {
	p.lock.Lock()
	defer p.lock.Unlock()

	l, found := p.mapLinks[id]
	if !found {
		return nil, false
	}
	delete(p.mapLinks, id)

	// 帮我把 l 从 linkedList 中移除
	for {
		v, ok := p.linkedList.PopFront()
		if !ok {
			break
		}
		if v.Info.ID == id {
			break
		}

		p.linkedList.PushBack(v)
	}

	zap.S().With(
		zap.Any("accountID", p.ID),
		zap.String("name", p.Name),
		zap.Any("linkID", id),
	).Infof("link removed from account")
	return l, true
}

func (p *Account) SendMsg(m *spgwpb.Msg) error {
	// 锁内只选 link + 调整轮询游标；l.SendMsg 移出锁外，避免 link 阻塞整个 Account
	p.lock.Lock()
	if p.linkedList.Size() == 0 {
		p.lock.Unlock()
		return fmt.Errorf("no link available, acc=%p", p)
	}
	l, _ := p.linkedList.PopFront()
	p.linkedList.PushBack(l)
	p.lock.Unlock()

	return l.SendMsg(m)
}
