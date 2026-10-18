package service

import (
	"errors"

	"github.com/jinzhu/copier"

	"github.com/morya/gommon/gostring"

	"github.com/morya/spgw/internal/consts"
	"github.com/morya/spgw/internal/daox"
	"github.com/morya/spgw/internal/global/app"
	"github.com/morya/spgw/internal/models"
	"github.com/morya/spgw/spgwpb"
)

func PutOAccount(req *models.ReqPutOAccount) error {
	var oa *models.OAccount
	var err error
	if req.ID != 0 {
		oa, err = daox.GetOAccountByID(uint(req.ID))
		if err != nil {
			return errors.New("oaccount not exists")
		}
	} else {
		oa = &models.OAccount{}
	}
	copier.Copy(oa, req)
	return daox.PutOAccount(oa)
}

func GetOAccountByID(id uint) (*models.OAccount, error) {
	return daox.GetOAccountByID(id)
}

func DeleteOAccountByID(id uint) error {
	return daox.DeleteOAccountByID(id)
}

func ListOAccount(req *models.ReqListOAccount) (int64, []*models.OAccount, error) {
	return daox.ListOAccount(req)
}

func ListOAccountLinks(oaccountID uint) ([]*models.LinkInfo, error) {
	return daox.ListOAccountLinks(oaccountID)
}

func PutOAccountLink(req *models.ReqPutLinkInfo) error {
	var link *models.LinkInfo
	var err error
	if req.ID != 0 {
		link, err = daox.GetLinkInfoByID(req.ID)
		if err != nil {
			return errors.New("link not exists")
		}
	} else {
		link = &models.LinkInfo{}
	}
	copier.Copy(link, req)
	if err := daox.PutLinkInfo(link); err != nil {
		return err
	}
	notifyOAgentReloadOLink()
	return nil
}

func GetOAccountLinkByID(id uint) (*models.LinkInfo, error) {
	return daox.GetLinkInfoByID(id)
}

func DeleteOAccountLinkByID(id uint) error {
	l, err := GetOAccountLinkByID(id)
	if err != nil {
		return err
	}

	if err := daox.DeleteLinkInfoByID(id); err != nil {
		return err
	}

	m := spgwpb.NewMsg(spgwpb.CmdID_CmdDeleteOLink, 0)
	m.Head.FromNodeType = spgwpb.NodeType_NodeTypePortal
	m.DeleteOlinkReq = &spgwpb.DeleteOLinkReq{
		AccountId: uint32(l.OAccountID),
		LinkId:    uint32(id),
	}

	app.NatsClient.Publish(consts.TopicOAgentSys, gostring.JsonEncode(m))

	return nil
}

func notifyOAgentReloadOLink() {
	m := spgwpb.NewMsg(spgwpb.CmdID_CmdReloadOLink, 0)
	m.Head.FromNodeType = spgwpb.NodeType_NodeTypePortal
	app.NatsClient.Publish("spgw.oagent-sys", gostring.JsonEncode(m))
}
