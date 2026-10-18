package router_helper

import (
	"github.com/gin-gonic/gin"

	"github.com/morya/spgw/internal/router/response"
)

// HandleWithModel 把请求体绑定+校验封装成 gin.HandlerFunc。
// 业务 handler 签名：func(c *gin.Context, req *T)，不返回 error。
func HandleWithModel[T any](handler func(c *gin.Context, req *T)) gin.HandlerFunc {
	return func(c *gin.Context) {
		p := new(T)

		if err := BindAndValid(c, p); err != nil {
			response.Error(c, err)
			return
		}

		handler(c, p)
	}
}
