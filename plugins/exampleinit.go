package plugins

import (
	_ "github.com/morya/spgw/plugins/example/routes"
	"github.com/morya/spgw/plugins/example/scheduler"
)

// Init 插件初始化，需要在 bootstrap.Init() 之后调用
func Init() {
	// 注册示例执行器
	scheduler.RegisterExampleExecutors()
}
