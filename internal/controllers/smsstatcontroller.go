package controllers

import (
	"github.com/gin-gonic/gin"

	"github.com/morya/spgw/internal/models"
	"github.com/morya/spgw/internal/service"
)

func NewSmsStatController() *SmsStatController {
	return &SmsStatController{}
}

type SmsStatController struct{}

// ListIStat 获取北向账号统计列表
// @Summary 获取北向账号统计列表
// @Tags sms
// @Accept json
// @Produce json
// @Param req query models.ReqListIStat false "查询参数"
// @Success 200 {object} app.JsonResponse{data=models.IStatListData} "成功"
// @Failure 400 {object} app.JsonResponse "参数错误"
// @Failure 500 {object} app.JsonResponse "服务器内部错误"
// @Security ApiKeyAuth
// @Router /sms/istat [get]
func (c *SmsStatController) ListIStat(ctx *gin.Context) {
	req := &models.ReqListIStat{}
	if err := models.BaseValidator.Check(ctx, req); err != nil {
		CommonResponse.FailAndAbort(ctx, err.Error(), err)
	}

	total, list, err := service.ListIStat(req)
	if err != nil {
		CommonResponse.FailAndAbort(ctx, err.Error(), err)
	}
	CommonResponse.Success(ctx, gin.H{
		"list":  list,
		"total": total,
	})
}

// ListOStat 获取南向账号统计列表
// @Summary 获取南向账号统计列表
// @Tags sms
// @Accept json
// @Produce json
// @Param req query models.ReqListOStat false "查询参数"
// @Success 200 {object} app.JsonResponse{data=models.OStatListData} "成功"
// @Failure 400 {object} app.JsonResponse "参数错误"
// @Failure 500 {object} app.JsonResponse "服务器内部错误"
// @Security ApiKeyAuth
// @Router /sms/ostat [get]
func (c *SmsStatController) ListOStat(ctx *gin.Context) {
	req := &models.ReqListOStat{}
	if err := models.BaseValidator.Check(ctx, req); err != nil {
		CommonResponse.FailAndAbort(ctx, err.Error(), err)
	}

	total, list, err := service.ListOStat(req)
	if err != nil {
		CommonResponse.FailAndAbort(ctx, err.Error(), err)
	}
	CommonResponse.Success(ctx, gin.H{
		"list":  list,
		"total": total,
	})
}
