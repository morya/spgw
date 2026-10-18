package iagent

import (
	"github.com/koss-null/list"

	"github.com/morya/spgw/internal/link"
)

type Account struct {
	ID   uint32
	Spid string

	Links list.Linked[*link.Link]
}

func NewAccount(id uint32, spid string) *Account {
	return &Account{
		ID:    id,
		Spid:  spid,
		Links: list.Linked[*link.Link]{},
	}
}

func (acc *Account) LinkCount() int {
	return acc.Links.Size()
}

func (acc *Account) AddLink(l *link.Link) {
	acc.Links.PushBack(l)
}

func (acc *Account) NextLink() *link.Link {
	if acc.Links.Size() == 0 {
		return nil
	}
	l, _ := acc.Links.PopFront()
	acc.Links.PushBack(l)

	return l
}

func (acc *Account) RemoveLinkByID(id uint32) bool {
	var found bool
	var keep []*link.Link
	for {
		l, ok := acc.Links.PopFront()
		if !ok {
			break
		}
		if l.Info.ID == id {
			found = true
		} else {
			keep = append(keep, l)
		}
	}
	for _, l := range keep {
		acc.Links.PushBack(l)
	}
	return found
}
