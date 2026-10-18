package controllers

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"

	"github.com/morya/spgw/internal/global/app"
	"github.com/morya/spgw/internal/models"
	"github.com/morya/spgw/internal/service"
)

type SmsSpBillController struct{}

func NewSmsSpBillController() *SmsSpBillController {
	return &SmsSpBillController{}
}

// ListSpBill 获取 SP 话单文件列表
// @Summary SP 话单文件列表
// @Tags sms
// @Accept json
// @Produce json
// @Param req query models.ReqListSpBill false "查询参数"
// @Success 200 {object} app.JsonResponse{data=models.SpBillListData} "成功"
// @Failure 400 {object} app.JsonResponse "参数错误"
// @Failure 500 {object} app.JsonResponse "服务器内部错误"
// @Security ApiKeyAuth
// @Router /sms/sp-bills [get]
func (c *SmsSpBillController) ListSpBill(ctx *gin.Context) {
	req := &models.ReqListSpBill{}
	if err := models.BaseValidator.Check(ctx, req); err != nil {
		CommonResponse.FailAndAbort(ctx, err.Error(), err)
	}

	total, list, err := service.ListSpBill(req)
	if err != nil {
		CommonResponse.FailAndAbort(ctx, err.Error(), err)
	}
	CommonResponse.Success(ctx, gin.H{
		"list":  list,
		"total": total,
	})
}

// DownloadSpBill 下载 SP 话单文件
// @Summary 下载 SP 话单文件
// @Tags sms
// @Accept json
// @Produce application/gzip
// @Param fileid query int true "话单文件ID"
// @Success 200 {file} file "话单文件(.gz)"
// @Failure 400 {object} app.JsonResponse "参数错误"
// @Failure 404 {object} app.JsonResponse "话单文件不存在"
// @Failure 500 {object} app.JsonResponse "服务器内部错误"
// @Security ApiKeyAuth
// @Router /sms/sp-bills/export [get]
func (c *SmsSpBillController) DownloadSpBill(ctx *gin.Context) {
	fileID := cast.ToUint(ctx.Query("fileid"))
	if fileID == 0 {
		CommonResponse.FailAndAbort(ctx, "话单文件ID不能为空", nil)
		return
	}

	file, err := service.GetSpBillByID(fileID)
	if err != nil {
		CommonResponse.FailAndAbort(ctx, "话单文件记录不存在", err)
		return
	}

	// 防止路径穿越：FileName 必须是纯文件名，不含目录分隔符或 ..
	if filepath.Base(file.FileName) != file.FileName {
		CommonResponse.FailAndAbort(ctx, "无效的话单文件名", nil)
		return
	}

	// 确认解析后的绝对路径仍位于话单根目录之下
	absBillDir, err := filepath.Abs(app.BillDir)
	if err != nil {
		CommonResponse.FailAndAbort(ctx, "解析话单根目录失败", err)
		return
	}
	absFull, err := filepath.Abs(filepath.Join(app.BillDir, file.FileName))
	if err != nil {
		CommonResponse.FailAndAbort(ctx, "解析话单路径失败", err)
		return
	}
	if !strings.HasPrefix(absFull, absBillDir+string(filepath.Separator)) {
		CommonResponse.FailAndAbort(ctx, "无效的话单路径", nil)
		return
	}

	// 文件必须存在且不是目录
	info, err := os.Stat(absFull)
	if err != nil {
		if os.IsNotExist(err) {
			CommonResponse.FailAndAbort(ctx, "话单文件不存在", err)
			return
		}
		CommonResponse.FailAndAbort(ctx, "读取话单文件失败", err)
		return
	}
	if info.IsDir() {
		CommonResponse.FailAndAbort(ctx, "话单路径指向目录", nil)
		return
	}

	// 触发浏览器下载，文件名用数据库记录的 FileName（已是 .gz 基名）
	ctx.FileAttachment(absFull, file.FileName)
}
