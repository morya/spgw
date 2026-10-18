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

func PutSpInfo(req *models.ReqPutSpInfo) error {
	spInfo := &models.SpInfo{}
	var err error
	if req.ID != 0 {
		spInfo, err = daox.GetSpInfoByID(req.ID)
		if err != nil {
			return errors.New("sp not exists")
		}
	}
	copier.Copy(spInfo, req)
	err = daox.PutSpInfo(spInfo)
	if err != nil {
		return err
	}

	// 通知 iagent 关停相关account
	m := spgwpb.NewMsg(spgwpb.CmdID_CmdReloadIAccount, 0)
	m.Head.FromNodeType = spgwpb.NodeType_NodeTypePortal
	m.ReloadIaccountReq = &spgwpb.ReloadIAccountReq{
		AccountId: uint32(spInfo.ID),
	}

	app.NatsClient.Publish(consts.TopicIAgentSys, gostring.JsonEncode(m))

	return nil
}

func GetSpInfoByID(id uint) (*models.SpInfo, error) {
	return daox.GetSpInfoByID(id)
}

func DeleteSpInfoByID(id uint) error {
	err := daox.DeleteSpInfoByID(id)
	if err != nil {
		return err
	}

	// 通知 iagent 关停相关account
	m := spgwpb.NewMsg(spgwpb.CmdID_CmdDeleteIAccount, 0)
	m.Head.FromNodeType = spgwpb.NodeType_NodeTypePortal

	app.NatsClient.Publish(consts.TopicIAgentSys, gostring.JsonEncode(m))

	return nil
}

func ListSpInfo(req *models.ReqListSpInfo) (int64, []*models.SpInfo, error) {
	total, list, err := daox.ListSpInfo(req)
	if err != nil {
		return 0, nil, err
	}

	return total, list, nil
}

func DropSpLink(linkID uint32) error {
	// 通知 iagent 关停相关account
	m := spgwpb.NewMsg(spgwpb.CmdID_CmdDeleteILink, linkID)
	m.Head.FromNodeType = spgwpb.NodeType_NodeTypePortal
	m.DeleteIlinkReq = &spgwpb.DeleteILinkReq{
		LinkId: linkID,
	}

	app.NatsClient.Publish(consts.TopicIAgentSys, gostring.JsonEncode(m))
	return nil
}
