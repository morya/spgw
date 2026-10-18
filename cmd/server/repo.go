package server

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/koss-null/list"
	"github.com/spf13/cast"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"

	"github.com/morya/gommon/gostring"

	"github.com/morya/spgw/internal/db"
	"github.com/morya/spgw/internal/models"
	"github.com/morya/spgw/spgwpb"
)

type Repo struct {
	mapIdToIAccounts   map[uint32]*IAccount // key=iAccID
	mapSpidToIAccounts map[string]*IAccount // key=spid

	mapIdToOAccounts  map[uint32]*OAccount // key=spid
	mapHlrToOAccounts map[string]*OAccount
	mapKeyToOLinks    map[uint32]*OLink // key=linkID

	ActiveOAccounts list.Linked[*OAccount] // 活跃的 oaccount 列表
	OAccountList    list.Linked[*OAccount]
}

func newRepo() *Repo {
	return &Repo{
		mapIdToIAccounts:   make(map[uint32]*IAccount),
		mapSpidToIAccounts: make(map[string]*IAccount),

		mapIdToOAccounts:  make(map[uint32]*OAccount),
		mapHlrToOAccounts: make(map[string]*OAccount),
		mapKeyToOLinks:    make(map[uint32]*OLink),

		OAccountList:    list.Linked[*OAccount]{},
		ActiveOAccounts: list.Linked[*OAccount]{},
	}
}

func (r *Repo) Load(c *clientv3.Client) error {
	if err := r.LoadAccounts(); err != nil {
		zap.S().Errorf("load iaccount error: %v", err)
		return err
	}

	if err := r.loadILinks(c); err != nil {
		zap.S().Errorf("load ilink error: %v", err)
	}
	if err := r.loadOLinks(c); err != nil {
		zap.S().Errorf("load olink error: %v", err)
	}

	for _, oa := range r.mapIdToOAccounts {
		if !oa.Empty() {
			r.ActiveOAccounts.PushBack(oa)
		}
	}

	return nil
}

func (r *Repo) LoadAccounts() error {
	if err := r.loadIAccounts(); err != nil {
		return err
	}
	if err := r.loadOAccounts(); err != nil {
		return err
	}
	return nil
}

func (r *Repo) FindOAccountByID(id uint32) *OAccount {
	if oa, ok := r.mapIdToOAccounts[id]; ok {
		return oa
	}
	return nil
}

func (r *Repo) FindIAccountByID(id uint32) *IAccount {
	if ia, ok := r.mapIdToIAccounts[id]; ok {
		return ia
	}
	return nil
}

func (r *Repo) loadIAccounts() error {
	var arr []*models.SpInfo
	err := db.MainDB.Find(&arr).Error
	if err != nil {
		return err
	}

	for _, item := range arr {
		ia := NewIAccount(item)
		r.mapIdToIAccounts[uint32(ia.ID)] = ia
		r.mapSpidToIAccounts[item.Spid] = ia
	}
	return nil
}

func (r *Repo) loadOAccounts() error {
	var arr []*models.OAccount
	err := db.MainDB.Model(&models.OAccount{}).Find(&arr).Error
	if err != nil {
		return err
	}

	for _, item := range arr {
		oa := NewOAccount(item)
		r.OAccountList.PushBack(oa)
		r.mapIdToOAccounts[oa.ID] = oa
		if oa.HlrCode != "" {
			r.mapHlrToOAccounts[oa.HlrCode] = oa
		}
	}
	return nil
}

func (r *Repo) loadILinks(c *clientv3.Client) error {
	reg := regexp.MustCompile(`/spgw/ilink-(\d+)-(\d+)`)

	resp, err := c.Get(context.Background(), "/spgw/ilink-", clientv3.WithPrefix())
	if err != nil {
		zap.S().Errorf("get ilink error: %v", err)
		return err
	}

	for _, v := range resp.Kvs {
		matches := reg.FindStringSubmatch(string(v.Key))
		if len(matches) != 3 {
			continue
		}

		iagentID, linkID := cast.ToUint(matches[1]), cast.ToUint(matches[2])

		info := &ILink{}
		gostring.JsonDecode(v.Value, info)

		if ia, ok := r.mapSpidToIAccounts[info.Spid]; !ok {
			zap.S().Errorf("iaccount %s not found", info.Spid)
			continue
		} else {
			ia.Append(info)
		}

		zap.S().With(
			zap.Any("iagentID", iagentID)).
			With(zap.Any("linkID", linkID)).
			With(zap.Any("info", info)).
			Debugf("ilink created")
	}
	return nil
}

func (r *Repo) loadOLinks(c *clientv3.Client) error {
	reg := regexp.MustCompile(`/spgw/olink-(\d+)-(\d+)`)

	resp, err := c.Get(context.Background(), "/spgw/olink-", clientv3.WithPrefix())
	if err != nil {
		zap.S().Errorf("get olink error: %v", err)
		return err
	}

	for _, v := range resp.Kvs {
		matches := reg.FindStringSubmatch(string(v.Key))
		if len(matches) != 3 {
			continue
		}

		oagentID, linkID := cast.ToUint(matches[1]), cast.ToUint(matches[2])

		info := &OLink{}
		gostring.JsonDecode(v.Value, info)

		if oa, ok := r.mapIdToOAccounts[info.OAccountID]; !ok {
			zap.S().Errorf("oaccount with id %d not found", info.OAccountID)
			continue
		} else {
			// oa, ok = r.mapHlrToOAccounts[oa.HlrCode]
			// if !ok {
			// 	zap.S().Errorf("oaccount with hlr %s not found", oa.HlrCode)
			// 	continue
			// }
			oa.Append(info)
		}
		r.mapKeyToOLinks[uint32(linkID)] = info

		zap.S().With(
			zap.Any("oagentID", oagentID)).
			With(zap.Any("linkID", linkID)).
			With(zap.Any("info", info)).
			Debugf("olink created")
	}
	return nil
}

// HLR HOME LOCATION REGISTER 归属地编码
func (r *Repo) parsePhoneToHLR(phoneNum string) (string, error) {
	// 去除`+`前缀，`国家码`前缀
	if len(phoneNum) == 0 {
		return "", fmt.Errorf("phone num is empty")
	}
	phoneNum = strings.TrimPrefix(phoneNum, "+86")
	phoneNum = strings.TrimPrefix(phoneNum, "0086")
	if len(phoneNum) != 11 {
		return "", fmt.Errorf("phone num %s is invalid", phoneNum)
	}
	// 归属地编码
	return phoneNum[3:7], nil
}

func (r *Repo) RouteByHlr(msg *spgwpb.Msg, req *spgwpb.SubmitReq) (*OAccount, error) {
	// 无锁 上层函数有锁
	var phone string
	if req.FeeTerminalId != "" {
		phone = req.FeeTerminalId
	} else {
		if len(req.DestTerminalId) == 0 {
			return nil, fmt.Errorf("dest terminal id is empty")
		}
		phone = req.DestTerminalId[0]
	}

	hlr, err := r.parsePhoneToHLR(phone)
	if err != nil {
		return nil, err
	}

	oa := r.mapHlrToOAccounts[hlr]
	if oa == nil {
		return nil, fmt.Errorf("no oaccount found for hlr %s", hlr)
	}
	return oa, nil
}

func (r *Repo) nextOAccount() *OAccount {
	if r.ActiveOAccounts.Size() == 0 {
		return nil
	}

	oa, _ := r.ActiveOAccounts.PopFront()
	defer r.ActiveOAccounts.PushBack(oa)
	return oa
}

func (r *Repo) RouteByWeight(msg *spgwpb.Msg, req *spgwpb.SubmitReq) (*OAccount, error) {
	// TODO: 权重路由需要根据权重发送量进行路由
	acc := r.nextOAccount()
	if acc == nil {
		return nil, fmt.Errorf("no oaccount found")
	}

	return acc, nil
}
