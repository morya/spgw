package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/morya/spgw/internal/global/app"
	"github.com/morya/spgw/internal/global/consts"
	"github.com/morya/spgw/internal/utils/common"
)

// JWTAuthMiddleware JWT认证中间件
func JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, err := common.GetAccessToken(c)
		if err != nil {
			app.ZapLog.Error("Get access token failed", zap.Error(err))
			// 401 未认证
			c.JSON(http.StatusUnauthorized, gin.H{"message": err.Error()})
			c.Abort()
			return
		}

		// 验证token
		claims, err := app.TokenService.ValidateTokenWithCache(tokenString)
		if err != nil {
			app.ZapLog.Error("Invalid token", zap.Error(err))
			// 401 未认证
			c.JSON(http.StatusUnauthorized, gin.H{"message": err.Error()})
			c.Abort()
			return
		}
		// 将用户信息存储到上下文中
		c.Set(consts.BindContextKeyName, claims)
		// 继续处理请求
		c.Next()
	}
}
