package middleware

import (
	"github.com/gin-gonic/gin"

	"github.com/morya/spgw/internal/global/app"
)

// CasbinMiddleware Casbin权限中间件
func CasbinMiddleware() gin.HandlerFunc {
	// return casbinhelper.GetCasbinService().CasbinMiddleware()
	return app.CasbinV2.CasbinMiddleware()
}
