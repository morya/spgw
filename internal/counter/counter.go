package counter

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/muesli/cache2go"
	"github.com/spf13/cast"
	"go.uber.org/zap"
	"gorm.io/gorm/clause"

	"github.com/morya/spgw/internal/db"
	"github.com/morya/spgw/internal/models"
)

var (
	iCounters *cache2go.CacheTable
	oCounters *cache2go.CacheTable
	nodeID    uint32
)

func init() {
	iCounters = cache2go.Cache("iCounters")
	iCounters.SetDataLoader(func(i1 any, i2 ...any) *cache2go.CacheItem {
		accID := cast.ToUint(i1)
		return cache2go.NewCacheItem(accID, time.Hour, &ICounter{
			ID:     accID,
			NodeID: nodeID,
		})
	})

	oCounters = cache2go.Cache("oCounters")
	oCounters.SetDataLoader(func(i1 any, i2 ...any) *cache2go.CacheItem {
		accID := cast.ToUint(i1)
		return cache2go.NewCacheItem(accID, time.Hour, &OCounter{
			ID:     accID,
			NodeID: nodeID,
		})
	})
}

func SetNodeID(id uint32) {
	nodeID = id
}

type ICounter struct {
	ID     uint   `json:"id" gorm:"primaryKey"`
	SpName string `json:"spName"`
	NodeID uint32 `json:"nodeID"`

	SubmitSuccCnt uint32
	SubmitFailCnt uint32
	ReportSuccCnt uint32
	ReportFailCnt uint32
}

func (c *ICounter) Incr(t *uint32) {
	atomic.AddUint32(t, 1)
}

type OCounter struct {
	ID     uint   `json:"id" gorm:"primaryKey"`
	SpName string `json:"spName"`
	NodeID uint32 `json:"nodeID"`

	SubmitSuccCnt uint32
	SubmitFailCnt uint32
	ReportSuccCnt uint32
	ReportFailCnt uint32
}

func (c *OCounter) Incr(t *uint32) {
	atomic.AddUint32(t, 1)
}

func incrICounter(id uint32, field func(*ICounter)) {
	v, err := iCounters.Value(id)
	if err != nil || v == nil {
		return
	}
	field(v.Data().(*ICounter))
}

func incrOCounter(id uint32, field func(*OCounter)) {
	v, err := oCounters.Value(id)
	if err != nil || v == nil {
		return
	}
	field(v.Data().(*OCounter))
}

func IncrISubmitSuccCnt(id uint32) {
	incrICounter(id, func(c *ICounter) { c.Incr(&c.SubmitSuccCnt) })
}

func IncrISubmitFailCnt(id uint32) {
	incrICounter(id, func(c *ICounter) { c.Incr(&c.SubmitFailCnt) })
}

func IncrIReportSuccCnt(id uint32) {
	incrICounter(id, func(c *ICounter) { c.Incr(&c.ReportSuccCnt) })
}

func IncrIReportFailCnt(id uint32) {
	incrICounter(id, func(c *ICounter) { c.Incr(&c.ReportFailCnt) })
}

func IncrOSubmitSuccCnt(id uint32) {
	incrOCounter(id, func(c *OCounter) { c.Incr(&c.SubmitSuccCnt) })
}

func IncrOSubmitFailCnt(id uint32) {
	incrOCounter(id, func(c *OCounter) { c.Incr(&c.SubmitFailCnt) })
}

func IncrOSReportSuccCnt(id uint32) {
	incrOCounter(id, func(c *OCounter) { c.Incr(&c.ReportSuccCnt) })
}

func IncrOSReportFailCnt(id uint32) {
	incrOCounter(id, func(c *OCounter) { c.Incr(&c.ReportFailCnt) })
}

func swapAndBuildIStat(c *ICounter) *models.IStat {
	ss := atomic.SwapUint32(&c.SubmitSuccCnt, 0)
	sf := atomic.SwapUint32(&c.SubmitFailCnt, 0)
	rs := atomic.SwapUint32(&c.ReportSuccCnt, 0)
	rf := atomic.SwapUint32(&c.ReportFailCnt, 0)
	if ss == 0 && sf == 0 && rs == 0 && rf == 0 {
		return nil
	}

	n := time.Now()

	return &models.IStat{
		CreatedAt:     n.Unix() / 60 * 60,
		UpdatedAt:     n.Unix() / 60 * 60,
		IAccountID:    uint32(c.ID),
		NodeID:        c.NodeID,
		SubmitSuccCnt: ss,
		SubmitFailCnt: sf,
		ReportSuccCnt: rs,
		ReportFailCnt: rf,
	}
}

func swapAndBuildOStat(c *OCounter) *models.OStat {
	ss := atomic.SwapUint32(&c.SubmitSuccCnt, 0)
	sf := atomic.SwapUint32(&c.SubmitFailCnt, 0)
	rs := atomic.SwapUint32(&c.ReportSuccCnt, 0)
	rf := atomic.SwapUint32(&c.ReportFailCnt, 0)
	if ss == 0 && sf == 0 && rs == 0 && rf == 0 {
		return nil
	}
	return &models.OStat{
		OAccountID:    uint32(c.ID),
		NodeID:        c.NodeID,
		SubmitSuccCnt: ss,
		SubmitFailCnt: sf,
		ReportSuccCnt: rs,
		ReportFailCnt: rf,
	}
}

func FlushToDB() error {
	if db.MainDB == nil {
		return fmt.Errorf("main db is nil")
	}

	var iStats []*models.IStat
	iCounters.Foreach(func(key any, item *cache2go.CacheItem) {
		c := item.Data().(*ICounter)
		if s := swapAndBuildIStat(c); s != nil {
			iStats = append(iStats, s)
		}
	})

	if len(iStats) > 0 {
		zap.S().Infof("flush %v istat to db", len(iStats))

		if err := db.MainDB.Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "iaccount_id"},
				{Name: "node_id"},
			},
			DoUpdates: clause.Assignments(map[string]any{
				"submit_succ_cnt": clause.Expr{SQL: "gw_istat.submit_succ_cnt + VALUES(submit_succ_cnt)"},
				"submit_fail_cnt": clause.Expr{SQL: "gw_istat.submit_fail_cnt + VALUES(submit_fail_cnt)"},
				"report_succ_cnt": clause.Expr{SQL: "gw_istat.report_succ_cnt + VALUES(report_succ_cnt)"},
				"report_fail_cnt": clause.Expr{SQL: "gw_istat.report_fail_cnt + VALUES(report_fail_cnt)"},
			}),
		}).Create(&iStats).Error; err != nil {
			return fmt.Errorf("upsert istat failed: %w", err)
		}
	}

	var oStats []*models.OStat
	oCounters.Foreach(func(key any, item *cache2go.CacheItem) {
		c := item.Data().(*OCounter)
		if s := swapAndBuildOStat(c); s != nil {
			oStats = append(oStats, s)
		}
	})

	if len(oStats) > 0 {
		zap.S().Infof("flush %v ostat to db", len(oStats))

		if err := db.MainDB.Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "oaccount_id"},
				{Name: "node_id"},
			},
			DoUpdates: clause.Assignments(map[string]any{
				"submit_succ_cnt": clause.Expr{SQL: "gw_ostat.submit_succ_cnt + VALUES(submit_succ_cnt)"},
				"submit_fail_cnt": clause.Expr{SQL: "gw_ostat.submit_fail_cnt + VALUES(submit_fail_cnt)"},
				"report_succ_cnt": clause.Expr{SQL: "gw_ostat.report_succ_cnt + VALUES(report_succ_cnt)"},
				"report_fail_cnt": clause.Expr{SQL: "gw_ostat.report_fail_cnt + VALUES(report_fail_cnt)"},
			}),
		}).Create(&oStats).Error; err != nil {
			return fmt.Errorf("upsert ostat failed: %w", err)
		}
	}

	return nil
}
