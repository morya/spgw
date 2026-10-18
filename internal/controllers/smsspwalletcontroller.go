package controllers

import (
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"

	"github.com/morya/spgw/internal/models"
	"github.com/morya/spgw/internal/service"
)

type SmsSpWalletController struct{}

func NewSmsSpWalletController() *SmsSpWalletController {
	return &SmsSpWalletController{}
}

// GetSpWallet 获取钱包信息
// @Summary 获取钱包信息
// @Tags sms
// @Accept json
// @Produce json
// @Param wallet_id path int true "钱包ID"
// @Success 200 {object} app.JsonResponse{data=models.SpWallet} "成功"
// @Failure 400 {object} app.JsonResponse "参数错误"
// @Failure 500 {object} app.JsonResponse "服务器内部错误"
// @Security ApiKeyAuth
// @Router /sms/wallet-detail/{wallet_id} [get]
func (c *SmsSpWalletController) GetSpWallet(ctx *gin.Context) {
	walletId := cast.ToUint(ctx.Param("wallet_id"))
	if walletId == 0 {
		CommonResponse.FailAndAbort(ctx, "ID不能为空", nil)
	}

	wallet, err := service.GetSpWalletByID(walletId)
	if err != nil {
		CommonResponse.FailAndAbort(ctx, err.Error(), err)
	}
	CommonResponse.Success(ctx, wallet)
}

// AdjustSpWallet 调整钱包余额
// @Summary 调整钱包余额（增加或扣减）
// @Tags sms
// @Accept json
// @Produce json
// @Param req body models.ReqAdjustSpWallet true "调整参数"
// @Success 200 {object} app.JsonResponse "成功"
// @Failure 400 {object} app.JsonResponse "参数错误"
// @Failure 500 {object} app.JsonResponse "服务器内部错误"
// @Security ApiKeyAuth
// @Router /sms/wallet-adjust [post]
func (cc *SmsSpWalletController) AdjustSpWallet(c *gin.Context) {
	req := &models.ReqAdjustSpWallet{}
	if err := models.BaseValidator.Check(c, req); err != nil {
		CommonResponse.FailAndAbort(c, err.Error(), err)
	}

	err := service.AdjustSpWallet(req)
	if err != nil {
		CommonResponse.FailAndAbort(c, err.Error(), err)
	}
	CommonResponse.Success(c, nil)
}

// ListSpWalletLog 获取钱包变动记录列表
// @Summary 获取钱包变动记录列表
// @Tags sms
// @Accept json
// @Produce json
// @Param req query models.ReqListSpWalletLog false "查询参数"
// @Success 200 {object} app.JsonResponse{data=models.SpWalletLogListData} "成功"
// @Failure 400 {object} app.JsonResponse "参数错误"
// @Failure 500 {object} app.JsonResponse "服务器内部错误"
// @Security ApiKeyAuth
// @Router /sms/wallet-log [get]
func (c *SmsSpWalletController) ListSpWalletLog(ctx *gin.Context) {
	req := &models.ReqListSpWalletLog{}
	if err := models.BaseValidator.Check(ctx, req); err != nil {
		CommonResponse.FailAndAbort(ctx, err.Error(), err)
	}
	if req.PageSize == -1 {
		req.PageSize = 1000
	}
	if req.Order == "" {
		req.Order = "id desc"
	}

	list := models.NewSpWalletLogList()
	total, err := list.GetTotal(ctx, req.Handle())
	if err != nil {
		CommonResponse.FailAndAbort(ctx, err.Error(), err)
	}
	err = list.Find(ctx, req.Paginate(), req.Handle())
	if err != nil {
		CommonResponse.FailAndAbort(ctx, err.Error(), err)
	}
	CommonResponse.Success(ctx, gin.H{
		"list":  list,
		"total": total,
	})
}
