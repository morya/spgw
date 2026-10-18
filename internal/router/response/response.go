package response

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/morya/gommon/gostring"

	"github.com/morya/spgw/internal/models"
)

// 主要做demo接口演示使用
func OK(c *gin.Context) {
	c.String(http.StatusOK, "ok")
}

// JSON 原样将obj编码为json送出 不包装任何层级
func JSON(c *gin.Context, obj any) {
	if v, ok := obj.(*models.RspBase); ok {
		v.Message = "Success"
		v.Timestamp = time.Now().Unix()
	}
	c.JSON(http.StatusOK, obj)
}

func Success(c *gin.Context) {
	SuccessWithMsg(c)
}

// 仅读取 msg 第一个参数
func SuccessWithMsg(c *gin.Context, msg ...string) {
	var m string
	if len(msg) == 0 {
		m = "Success"
	} else {
		m = msg[0]
	}
	rsp := &models.RspBase{
		Message:   m,
		Timestamp: time.Now().Unix(),
	}
	c.JSON(http.StatusOK, rsp)
}

// SuccessWithData 直接把data放到json的data字段
func SuccessWithData(c *gin.Context, data any) {
	rsp := &models.RspSuccess{
		RspBase: models.RspBase{
			Message:   "Success",
			Timestamp: time.Now().Unix(),
		},
		Data: data,
	}
	c.JSON(http.StatusOK, rsp)
}

// wrap https://github.com/wI2L/jettison/issues/13
func SuccessWithCustomizedJson(c *gin.Context, data any) {
	rsp := &models.RspSuccess{
		RspBase: models.RspBase{
			Message:   "Success",
			Timestamp: time.Now().Unix(),
		},
		Data: data,
	}
	s := gostring.JsonEncode(rsp)
	c.Data(http.StatusOK, "application/json", s)
}

func SuccessPagedResult(c *gin.Context, p *models.RspPaginator) {
	rsp := &models.RspSuccess{
		RspBase: models.RspBase{
			Message:   "Success",
			Timestamp: time.Now().Unix(),
		},
		Data: p,
	}

	c.JSON(http.StatusOK, rsp)
}

// wrap https://github.com/wI2L/jettison/issues/13
func SuccessPagedResultWithCustomizedJson(c *gin.Context, p *models.RspPaginator) {
	rsp := &models.RspSuccess{
		RspBase: models.RspBase{
			Message:   "Success",
			Timestamp: time.Now().Unix(),
		},
		Data: p,
	}
	s := gostring.JsonEncode(rsp)
	c.Data(http.StatusOK, "application/json", s)
}
