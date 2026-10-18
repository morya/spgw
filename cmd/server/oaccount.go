package server

import (
	"github.com/koss-null/list"

	"github.com/morya/spgw/internal/models"
)

type OLink struct {
	models.LinkInfo
}

type OAccount struct {
	models.OAccount
	links list.Linked[*OLink]
}

func NewOAccount(acc *models.OAccount) *OAccount {
	return &OAccount{
		OAccount: *acc,
	}
}

func (oa *OAccount) Append(link *OLink) {
	oa.links.PushBack(link)
}

func (oa *OAccount) Remove(link *OLink) bool {
	var found bool
	var keep []*OLink
	for {
		l, ok := oa.links.PopFront()
		if !ok {
			break
		}
		if l.ID == link.ID {
			found = true
		} else {
			keep = append(keep, l)
		}
	}
	for _, l := range keep {
		oa.links.PushBack(l)
	}
	return found
}

func (oa *OAccount) Empty() bool {
	return oa.links.Size() == 0
}

func (oa *OAccount) NextLink() *OLink {
	if oa.links.Size() == 0 {
		return nil
	}
	l, _ := oa.links.PopFront()
	oa.links.PushBack(l)
	return l
}
