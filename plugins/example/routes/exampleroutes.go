package routes

import (
	"github.com/gin-gonic/gin"

	"github.com/morya/spgw/internal/global/app"
	"github.com/morya/spgw/internal/middleware"
	"github.com/morya/spgw/internal/utils/ginhelper"
	"github.com/morya/spgw/plugins/example/controllers"
)

func Init() {
	exampleControllers := controllers.NewExampleController()
	ginhelper.RegisterPluginRoutes(func(engine *gin.Engine) {
		// 示例插件路由组
		example := engine.Group("/api/plugins/example")
		example.Use(middleware.JWTAuthMiddleware())     // 认证中间件
		example.Use(middleware.DemoAccountMiddleware()) // 添加演示账号中间件
		example.Use(middleware.CasbinMiddleware())      // 权限中间件
		{
			// 创建示例
			example.POST("/add", exampleControllers.Create)
			// 更新示例
			example.PUT("/edit", exampleControllers.Update)
			// 删除示例
			example.DELETE("/delete", exampleControllers.Delete)
			// 示例列表（分页查询）
			example.GET("/list", exampleControllers.List)
			// 根据ID获取示例信息
			example.GET("/:id", exampleControllers.GetByID)
		}

		app.ZapLog.Info("示例插件路由注册成功")
	})
}
