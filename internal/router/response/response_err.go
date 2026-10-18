package response

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/morya/spgw/internal/consts"
	"github.com/morya/spgw/internal/models"
)

// 使用这个函数，而不是直接在 Handler 里面 return err
// 主要原因是，这里会返回 http.StatusOK 前端能够正常接收到 message 信息
func Error(c *gin.Context, err error) {
	if err == nil {
		err = errors.New("nil error")
	}
	rsp := &models.RspFailure{
		RspBase: models.RspBase{
			Code:    consts.ErrRestCodeErrParam,
			Message: err.Error(),
		},
	}

	c.JSON(http.StatusOK, rsp)
}

func ErrorString(c *gin.Context, message string) {
	rsp := &models.RspFailure{
		RspBase: models.RspBase{
			Code:    consts.ErrRestCodeErrParam,
			Message: message,
		},
	}

	c.JSON(http.StatusOK, rsp)
}

func ErrorWithData(c *gin.Context, errcode int, msg string, data any) {
	c.JSON(http.StatusOK, &models.RspFailure{
		RspBase: models.RspBase{
			Code:    errcode,
			Message: msg,
		},
		Data: data,
	})
}

func ErrorWithCode(c *gin.Context, errcode int, message string) {
	rsp := &models.RspFailure{
		RspBase: models.RspBase{
			Code:    errcode,
			Message: message,
		},
	}

	c.JSON(http.StatusOK, rsp)
}

func ErrorWithHttpCode(c *gin.Context, httpCode, errcode int, message string) {
	rsp := &models.RspFailure{
		RspBase: models.RspBase{
			Code:    errcode,
			Message: message,
		},
	}

	c.JSON(httpCode, rsp)
}
