package ss

import (
	"context"

	"go.uber.org/zap"

	"github.com/morya/spgw/spgwpb"
)

type App struct{}

func NewApp() *App {
	return &App{}
}

func (a *App) Greeting(ctx context.Context, req *spgwpb.HelloReq, rsp *spgwpb.HelloRsp) error {
	zap.S().Infof("Hello %s", req.Name)
	rsp.Msg = "Hello " + req.Name
	return nil
}
