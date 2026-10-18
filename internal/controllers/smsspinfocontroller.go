package controllers

import (
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"

	"github.com/morya/spgw/internal/models"
	"github.com/morya/spgw/internal/service"
)

type SmsSpController struct{}

func NewSmsSpController() *SmsSpController {
	return &SmsSpController{}
}

// PutSpInfo 添加或更新短信配置信息
// @Summary 添加或更新短信配置信息
// @Tags sms
// @Accept json
// @Produce json
// @Param req body models.ReqPutSpInfo true "短信配置信息"
// @Success 200 {object} app.JsonResponse "成功"
// @Failure 400 {object} app.JsonResponse "参数错误"
// @Failure 500 {object} app.JsonResponse "服务器内部错误"
// @Security ApiKeyAuth
// @Router /sms/spinfo [put]
func (c *SmsSpController) PutSpInfo(ctx *gin.Context) {
	req := &models.ReqPutSpInfo{}
	if err := models.BaseValidator.Check(ctx, req); err != nil {
		CommonResponse.FailAndAbort(ctx, err.Error(), err)
	}

	err := service.PutSpInfo(req)
	if err != nil {
		CommonResponse.FailAndAbort(ctx, err.Error(), err)
	}
	CommonResponse.Success(ctx, nil)
}

// GetSpInfo 获取短信配置信息
// @Summary 获取短信配置信息
// @Tags sms
// @Accept json
// @Produce json
// @Param id path int true "短信配置ID"
// @Success 200 {object} app.JsonResponse{data=models.SpInfo} "成功"
// @Failure 400 {object} app.JsonResponse "参数错误"
// @Failure 500 {object} app.JsonResponse "服务器内部错误"
// @Security ApiKeyAuth
// @Router /sms/sp/{id} [get]
func (c *SmsSpController) GetSpInfo(ctx *gin.Context) {
	id := cast.ToUint(ctx.Param("id"))
	if id == 0 {
		CommonResponse.FailAndAbort(ctx, "ID不能为空", nil)
	}

	spInfo, err := service.GetSpInfoByID(id)
	if err != nil {
		CommonResponse.FailAndAbort(ctx, err.Error(), err)
	}
	CommonResponse.Success(ctx, spInfo)
}

// DeleteSpInfo 删除短信配置信息
// @Summary 删除短信配置信息
// @Tags sms
// @Accept json
// @Produce json
// @Param id path int true "短信配置ID"
// @Success 200 {object} app.JsonResponse "成功"
// @Failure 400 {object} app.JsonResponse "参数错误"
// @Failure 500 {object} app.JsonResponse "服务器内部错误"
// @Security ApiKeyAuth
// @Router /sms/sp/{id} [delete]
func (c *SmsSpController) DeleteSpInfo(ctx *gin.Context) {
	id := cast.ToUint(ctx.Param("id"))
	if id == 0 {
		CommonResponse.FailAndAbort(ctx, "ID不能为空", nil)
	}

	err := service.DeleteSpInfoByID(id)
	if err != nil {
		CommonResponse.FailAndAbort(ctx, err.Error(), err)
	}
	CommonResponse.Success(ctx, nil)
}

// ListSpInfo 获取短信配置信息
// @Summary 获取短信配置信息
// @Tags sms
// @Accept json
// @Produce json
// @Param req query models.ReqListSpInfo false "查询参数"
// @Success 200 {object} app.JsonResponse{data=models.SpInfoListData} "成功"
// @Failure 400 {object} app.JsonResponse "参数错误"
// @Failure 500 {object} app.JsonResponse "服务器内部错误"
// @Security ApiKeyAuth
// @Router /sms/spinfos [get]
func (c *SmsSpController) ListSpInfo(ctx *gin.Context) {
	req := &models.ReqListSpInfo{}
	if err := models.BaseValidator.Check(ctx, req); err != nil {
		CommonResponse.FailAndAbort(ctx, err.Error(), err)
	}

	total, spInfoList, err := service.ListSpInfo(req)
	if err != nil {
		CommonResponse.FailAndAbort(ctx, err.Error(), err)
	}
	CommonResponse.Success(ctx, gin.H{
		"list":  spInfoList,
		"total": total,
	})
}
