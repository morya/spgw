package controllers

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/morya/gommon/gostring"

	"github.com/morya/spgw/internal/global/app"
	"github.com/morya/spgw/internal/models"
	"github.com/morya/spgw/internal/service"
)

// ListSpLinks 获取SP 接口列表
// @Summary 获取SP 当前活跃接口列表，无需分页
// @Tags sms
// @Accept json
// @Produce json
// @Success 200 {object} app.JsonResponse{data=models.LinkListData} "成功"
// @Failure 400 {object} app.JsonResponse "参数错误"
// @Failure 500 {object} app.JsonResponse "服务器内部错误"
// @Security ApiKeyAuth
// @Router /sms/sp-links/{iaccount_id} [get]
func (c *SmsSpController) ListSpLinks(ctx *gin.Context) {
	iaccountID := cast.ToUint32(ctx.Param("iaccount_id"))
	if iaccountID == 0 {
		CommonResponse.FailAndAbort(ctx, "参数iaccount_id错误", nil)
		return
	}

	arr := []*models.LinkInfo{}

	v := app.EtcdClient
	resp, err := v.Get(context.Background(), "/spgw/ilink-", clientv3.WithPrefix())
	if err != nil {
		CommonResponse.FailAndAbort(ctx, "获取SP接口列表错误", err)
		return
	}

	for _, v := range resp.Kvs {
		info := &models.LinkInfo{}
		gostring.JsonDecode(v.Value, info)

		if info.IAccountID != iaccountID {
			continue
		}
		arr = append(arr, info)
	}

	CommonResponse.Success(ctx, gin.H{
		"list":  arr,
		"total": len(arr),
	})
}

// DropSpLink 强制断开SP接口
// @Summary 强制断开SP接口
// @Tags sms
// @Accept json
// @Produce json
// @Param link_id path int true "SP接口ID"
// @Success 200 {object} app.JsonResponse "成功"
// @Failure 400 {object} app.JsonResponse "参数错误"
// @Failure 500 {object} app.JsonResponse "服务器内部错误"
// @Security ApiKeyAuth
// @Router /sms/sp-links/{link_id} [delete]
func (c *SmsSpController) DropSpLink(ctx *gin.Context) {
	linkID := cast.ToUint32(ctx.Param("link_id"))
	if linkID == 0 {
		CommonResponse.FailAndAbort(ctx, "参数link_id错误", nil)
		return
	}

	err := service.DropSpLink(linkID)
	if err != nil {
		CommonResponse.FailAndAbort(ctx, "断开SP接口错误", err)
		return
	}

	CommonResponse.Success(ctx)
}
