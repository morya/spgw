package server

import (
	"fmt"

	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"

	"github.com/morya/spgw/internal/counter"
	"github.com/morya/spgw/internal/gconfig"
	"github.com/morya/spgw/spgwpb"
)

type Router struct {
	mainRepo *Repo
}

func newRouter() *Router {
	return &Router{
		mainRepo: newRepo(),
	}
}

func (r *Router) Reload(c *clientv3.Client) error {
	back := newRepo()
	if err := back.Load(c); err != nil {
		zap.S().Errorf("load back repo error: %v", err)
		return err
	}
	r.mainRepo = back
	return nil
}

func (r *Router) FindOAccountByID(id uint32) *OAccount {
	return r.mainRepo.FindOAccountByID(id)
}

func (r *Router) FindIAccountByID(id uint32) *IAccount {
	return r.mainRepo.FindIAccountByID(id)
}

func (r *Router) IncrISubmitFailCnt(iAccID uint32) {
	counter.IncrISubmitFailCnt(iAccID)
}

func (r *Router) IncrISubmitSuccCnt(iAccID uint32) {
	counter.IncrISubmitSuccCnt(iAccID)
}

func (r *Router) IncrIReportSuccCnt(iAccID uint32) {
	counter.IncrIReportSuccCnt(iAccID)
}

func (r *Router) IncrIReportFailCnt(iAccID uint32) {
	counter.IncrIReportFailCnt(iAccID)
}

func (r *Router) IncrOSubmitSuccCnt(oAccID uint32) {
	counter.IncrOSubmitSuccCnt(oAccID)
}

func (r *Router) IncrOSubmitFailCnt(oAccID uint32) {
	counter.IncrOSubmitFailCnt(oAccID)
}

func (r *Router) IncrOSReportSuccCnt(oAccID uint32) {
	counter.IncrOSReportSuccCnt(oAccID)
}

func (r *Router) IncrOSReportFailCnt(oAccID uint32) {
	counter.IncrOSReportFailCnt(oAccID)
}

func (r *Router) Route(m *spgwpb.Msg, req *spgwpb.SubmitReq) (acc *OAccount, err error) {
	switch gconfig.Cfg.Server.RouteMode {
	case 1:
		acc, err = r.mainRepo.RouteByWeight(m, req)
	case 2:
		acc, err = r.mainRepo.RouteByHlr(m, req)
	default:
		return nil, fmt.Errorf("route mode %d not supported", gconfig.Cfg.Server.RouteMode)
	}

	return acc, err
}
