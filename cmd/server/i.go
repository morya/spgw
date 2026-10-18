package server

import (
	"github.com/koss-null/list"

	"github.com/morya/spgw/internal/models"
)

type ILink struct {
	models.LinkInfo
}

type IAccount struct {
	models.SpInfo

	Links list.Linked[*ILink]
}

func NewIAccount(a *models.SpInfo) *IAccount {
	return &IAccount{
		SpInfo: *a,
	}
}

func (v *IAccount) Append(l *ILink) error {
	v.Links.PushBack(l)
	return nil
}

// Remove 删除链接
func (v *IAccount) Remove(l *ILink) {
	var keep []*ILink
	for {
		link, ok := v.Links.PopFront()
		if !ok {
			break
		}
		if link.ID != l.ID {
			keep = append(keep, link)
		}
	}
	for _, link := range keep {
		v.Links.PushBack(link)
	}
}
