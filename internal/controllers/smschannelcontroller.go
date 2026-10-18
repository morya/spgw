package controllers

import (
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"

	"github.com/morya/spgw/internal/models"
	"github.com/morya/spgw/internal/service"
)

func NewSmsChannelController() *SmsChannelController {
	return &SmsChannelController{}
}

type SmsChannelController struct{}

// @Summary PutOAccount Upsert通道账号
// @Tags sms
// @Accept json
// @Produce json
// @Param id path int true "OAccount ID"
// @Success 200 {object} app.JsonResponse "成功"
// @Failure 400 {object} app.JsonResponse "参数错误"
// @Failure 500 {object} app.JsonResponse "服务器内部错误"
// @Security ApiKeyAuth
// @Router /sms/channel [put]
// @Param req body models.ReqPutOAccount true "OAccount信息"
func (c *SmsChannelController) PutOAccount(ctx *gin.Context) {
	req := &models.ReqPutOAccount{}
	if err := models.BaseValidator.Check(ctx, req); err != nil {
		CommonResponse.FailAndAbort(ctx, err.Error(), err)
	}

	err := service.PutOAccount(req)
	if err != nil {
		CommonResponse.FailAndAbort(ctx, err.Error(), err)
	}
	CommonResponse.Success(ctx, nil)
}

// @Summary GetOAccount 获取通道账号
// @Tags sms
// @Accept json
// @Produce json
// @Param id path int true "OAccount ID"
// @Success 200 {object} app.JsonResponse{data=models.OAccount} "成功"
// @Failure 400 {object} app.JsonResponse "参数错误"
// @Failure 500 {object} app.JsonResponse "服务器内部错误"
// @Security ApiKeyAuth
// @Router /sms/channel/{id} [get]
func (c *SmsChannelController) GetOAccount(ctx *gin.Context) {
	id := cast.ToUint(ctx.Param("id"))
	if id == 0 {
		CommonResponse.FailAndAbort(ctx, "ID不能为空", nil)
	}

	oa, err := service.GetOAccountByID(id)
	if err != nil {
		CommonResponse.FailAndAbort(ctx, err.Error(), err)
	}
	CommonResponse.Success(ctx, oa)
}

// @Summary DeleteOAccount 删除通道账号
// @Tags sms
// @Accept json
// @Produce json
// @Param id path int true "OAccount ID"
// @Success 200 {object} app.JsonResponse "成功"
// @Failure 400 {object} app.JsonResponse "参数错误"
// @Failure 500 {object} app.JsonResponse "服务器内部错误"
// @Security ApiKeyAuth
// @Router /sms/channel/{id} [delete]
func (c *SmsChannelController) DeleteOAccount(ctx *gin.Context) {
	id := cast.ToUint(ctx.Param("id"))
	if id == 0 {
		CommonResponse.FailAndAbort(ctx, "ID不能为空", nil)
	}

	err := service.DeleteOAccountByID(id)
	if err != nil {
		CommonResponse.FailAndAbort(ctx, err.Error(), err)
	}
	CommonResponse.Success(ctx, nil)
}

// ListOAccount 获取OAccount列表
// @Summary 获取OAccount列表
// @Tags sms
// @Accept json
// @Produce json
// @Param req query models.ReqListOAccount true "OAccount列表查询参数"
// @Success 200 {object} app.JsonResponse{data=models.OAccountWithConnsListData} "成功"
// @Failure 400 {object} app.JsonResponse "参数错误"
// @Failure 500 {object} app.JsonResponse "服务器内部错误"
// @Security ApiKeyAuth
// @Router /sms/channels [get]
func (c *SmsChannelController) ListOAccount(ctx *gin.Context) {
	var req models.ReqListOAccount
	if err := req.Validate(ctx); err != nil {
		CommonResponse.FailAndAbort(ctx, err.Error(), err)
	}

	if req.PageSize == -1 {
		req.PageSize = 1000
	}

	if req.Order == "" {
		req.Order = "id desc"
	}

	list := models.NewOAccountList()
	total, err := list.GetTotal(ctx, req.Handle())
	if err != nil {
		CommonResponse.FailAndAbort(ctx, err.Error(), err)
	}
	err = list.Find(ctx, req.Paginate(), req.Handle())
	if err != nil {
		CommonResponse.FailAndAbort(ctx, err.Error(), err)
	}

	// 收集所有 OAccount ID
	var ids []uint32
	for _, oa := range list {
		ids = append(ids, oa.ID)
	}

	// 查询 LinkInfo 连接数
	totalMap, activeMap, err := models.GetLinkCounts(ctx, ids)
	if err != nil {
		CommonResponse.FailAndAbort(ctx, err.Error(), err)
	}

	// 拼装响应
	result := make([]models.OAccountWithConns, len(list))
	for i, oa := range list {
		result[i] = models.OAccountWithConns{
			OAccount:    oa,
			TotalConns:  totalMap[oa.ID],
			ActiveConns: activeMap[oa.ID],
		}
	}

	CommonResponse.Success(ctx, gin.H{
		"list":  result,
		"total": total,
	})
}

// ListOAccountLinks 获取下游渠道链路列表
// @Summary 获取下游渠道链路列表
// @Tags sms
// @Accept json
// @Produce json
// @Param oaccount_id path int true "下游渠道ID"
// @Success 200 {object} app.JsonResponse{data=models.LinkListData} "成功"
// @Failure 400 {object} app.JsonResponse "参数错误"
// @Failure 500 {object} app.JsonResponse "服务器内部错误"
// @Security ApiKeyAuth
// @Router /sms/channel-links/{oaccount_id} [get]
func (c *SmsChannelController) ListOAccountLinks(ctx *gin.Context) {
	oaccountID := cast.ToUint(ctx.Param("oaccount_id"))
	if oaccountID == 0 {
		CommonResponse.FailAndAbort(ctx, "OAccount ID不能为空", nil)
	}

	links, err := service.ListOAccountLinks(oaccountID)
	if err != nil {
		CommonResponse.FailAndAbort(ctx, err.Error(), err)
	}
	CommonResponse.Success(ctx, gin.H{
		"list":  links,
		"total": len(links),
	})
}

// ListOAccountLinks 获取下游渠道链路列表
// @Summary 获取下游渠道链路列表
// @Tags sms
// @Accept json
// @Produce json
// @Param oaccount_id path int true "下游渠道ID"
// @Success 200 {object} app.JsonResponse{data=models.LinkDetailData} "成功"
// @Failure 400 {object} app.JsonResponse "参数错误"
// @Failure 500 {object} app.JsonResponse "服务器内部错误"
// @Security ApiKeyAuth
// @Router /sms/channel-links-detail/{link_id} [get]
func (c *SmsChannelController) GetOAccountLinkDetail(ctx *gin.Context) {
	linkID := cast.ToUint(ctx.Param("link_id"))
	if linkID == 0 {
		CommonResponse.FailAndAbort(ctx, "链路ID不能为空", nil)
	}

	link, err := service.GetOAccountLinkByID(linkID)
	if err != nil {
		CommonResponse.FailAndAbort(ctx, err.Error(), err)
	}
	CommonResponse.Success(ctx, gin.H{
		"link": link,
	})
}

// AddOAccountLink 新增下游渠道链路
// @Summary 新增下游渠道链路
// @Tags sms
// @Accept json
// @Produce json
// @Param req body models.ReqPutLinkInfo true "链路信息"
// @Success 200 {object} app.JsonResponse "成功"
// @Failure 400 {object} app.JsonResponse "参数错误"
// @Failure 500 {object} app.JsonResponse "服务器内部错误"
// @Security ApiKeyAuth
// @Router /sms/channel-links [post]
func (c *SmsChannelController) AddOAccountLink(ctx *gin.Context) {
	req := &models.ReqPutLinkInfo{}
	if err := models.BaseValidator.Check(ctx, req); err != nil {
		CommonResponse.FailAndAbort(ctx, err.Error(), err)
	}
	req.ID = 0
	if err := service.PutOAccountLink(req); err != nil {
		CommonResponse.FailAndAbort(ctx, err.Error(), err)
	}
	CommonResponse.Success(ctx, nil)
}

// EditOAccountLink 编辑下游渠道链路
// @Summary 编辑下游渠道链路
// @Tags sms
// @Accept json
// @Produce json
// @Param link_id path int true "链路ID"
// @Param req body models.ReqPutLinkInfo true "链路信息"
// @Success 200 {object} app.JsonResponse "成功"
// @Failure 400 {object} app.JsonResponse "参数错误"
// @Failure 500 {object} app.JsonResponse "服务器内部错误"
// @Security ApiKeyAuth
// @Router /sms/channel-links/{link_id} [put]
func (c *SmsChannelController) EditOAccountLink(ctx *gin.Context) {
	linkID := cast.ToUint(ctx.Param("link_id"))
	if linkID == 0 {
		CommonResponse.FailAndAbort(ctx, "链路ID不能为空", nil)
	}
	req := &models.ReqPutLinkInfo{}
	if err := models.BaseValidator.Check(ctx, req); err != nil {
		CommonResponse.FailAndAbort(ctx, err.Error(), err)
	}
	req.ID = linkID
	if err := service.PutOAccountLink(req); err != nil {
		CommonResponse.FailAndAbort(ctx, err.Error(), err)
	}
	CommonResponse.Success(ctx, nil)
}

// DropOAccountLink 删除下游渠道链路
// @Summary 删除下游渠道链路
// @Tags sms
// @Accept json
// @Produce json
// @Param link_id path int true "链路ID"
// @Success 200 {object} app.JsonResponse "成功"
// @Failure 400 {object} app.JsonResponse "参数错误"
// @Failure 500 {object} app.JsonResponse "服务器内部错误"
// @Security ApiKeyAuth
// @Router /sms/channel-links/{link_id} [delete]
func (c *SmsChannelController) DropOAccountLink(ctx *gin.Context) {
	linkID := cast.ToUint(ctx.Param("link_id"))
	if linkID == 0 {
		CommonResponse.FailAndAbort(ctx, "链路ID不能为空", nil)
	}
	if err := service.DeleteOAccountLinkByID(linkID); err != nil {
		CommonResponse.FailAndAbort(ctx, err.Error(), err)
	}
	CommonResponse.Success(ctx, nil)
}
