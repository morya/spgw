package main

import (
	"fmt"
	"os"
	"slices"
	"sort"
	"time"
)

func WriteResult(cfg *Config, t *Tracker, startedAt, sendEndAt, endAt time.Time) error {
	s := t.Snapshot()

	f, err := os.Create(cfg.ResultFile)
	if err != nil {
		return err
	}
	defer f.Close()

	w := func(format string, a ...any) {
		fmt.Fprintf(f, format, a...)
	}

	w("=== 运行参数 ===\n")
	w("protocol      : %s\n", cfg.Protocol)
	w("host          : %s\n", cfg.Host)
	w("spid          : %s\n", cfg.Spid)
	w("phone range   : %d ~ %d  (count=%d)\n", cfg.PhoneFrom, cfg.PhoneTo, cfg.PhoneTo-cfg.PhoneFrom+1)
	w("target total  : %d\n", cfg.Total)
	w("rate          : %d tps\n", cfg.Rate)
	w("src-id        : %s\n", cfg.SrcID)
	w("service-id    : %s\n", cfg.ServiceID)
	w("fee-type      : %s\n", cfg.FeeType)
	w("sign          : %s\n", cfg.Sign)
	w("content       : %s\n", cfg.Content)
	w("start         : %s\n", startedAt.Format("2006-01-02 15:04:05"))
	w("send end      : %s\n", sendEndAt.Format("2006-01-02 15:04:05"))
	w("end           : %s\n", endAt.Format("2006-01-02 15:04:05"))
	w("send elapsed  : %s\n", sendEndAt.Sub(startedAt).Round(time.Millisecond))
	w("total elapsed : %s\n", endAt.Sub(startedAt).Round(time.Millisecond))

	w("\n=== 发送 ===\n")
	w("submit_req sent : %d\n", s.Sent)
	w("send error      : %d\n", s.SendErr)
	if d := sendEndAt.Sub(startedAt).Seconds(); d > 0 {
		w("actual TPS      : %.2f\n", float64(s.Sent)/d)
	}

	w("\n=== deliver 统计 ===\n")
	w("deliver_req 接收 : %d\n", s.DeliverReqTotal)
	w("deliver_ack 回送 : %d\n", s.DeliverAckSent)

	w("\n=== submit_ack 统计 ===\n")
	w("received        : %d\n", s.AckTotal)
	w("matched seq     : %d   (与 submit_req 一一对应)\n", s.AckMatched)
	w("unmatched seq   : %d\n", s.AckUnmatch)
	w("missing         : %d   (超时未收到 ack 的 seq)\n", s.AckMissing)
	w("latency p50/p95/p99 : %s / %s / %s\n",
		s.AckLatencyP50.Round(time.Millisecond),
		s.AckLatencyP95.Round(time.Millisecond),
		s.AckLatencyP99.Round(time.Millisecond))

	w("\nack result 分布（按错误码）：\n")
	codes := make([]uint32, 0, len(s.AckResultDist))
	var ackResultSum uint64
	for k, v := range s.AckResultDist {
		codes = append(codes, k)
		ackResultSum += v
	}
	slices.Sort(codes)
	w("  (total=%d, ackTotal=%d, 一致性=%t)\n", ackResultSum, s.AckTotal, ackResultSum == s.AckTotal)
	for _, code := range codes {
		w("  %-6d %-20s : %d\n", code, ackCodeText(code), s.AckResultDist[code])
	}

	w("\nack 与号码对账（按号码前缀聚合）：\n")
	prefixes := make([]string, 0, len(s.PrefixDist))
	for k := range s.PrefixDist {
		prefixes = append(prefixes, k)
	}
	sort.Strings(prefixes)
	for _, p := range prefixes {
		pc := s.PrefixDist[p]
		w("  %s (%d 条) : ok=%d err=%d\n", p, pc.Total, pc.OK, pc.Err)
	}

	w("\n=== report (deliver_req with is_report=1) 统计 ===\n")
	w("received           : %d\n", s.RptTotal)
	w("matched submit     : %d\n", s.RptMatched)
	w("unmatched / 孤儿   : %d\n", s.RptUnmatch)
	w("ok-ack 但未收 report : %d\n", s.RptMissing)
	w("latency p50/p95/p99 (submit→report) : %s / %s / %s\n",
		s.RptLatencyP50.Round(time.Millisecond),
		s.RptLatencyP95.Round(time.Millisecond),
		s.RptLatencyP99.Round(time.Millisecond))

	w("\nstat 字段分布：\n")
	stats := make([]string, 0, len(s.RptStatDist))
	for k := range s.RptStatDist {
		stats = append(stats, k)
	}
	sort.Strings(stats)
	for _, st := range stats {
		w("  %-10s : %d\n", st, s.RptStatDist[st])
	}

	w("\n=== 异常 ===\n")
	w("TCP 中断重连次数 : %d\n", s.Reconnects)
	w("解码失败计数     : %d\n", s.DecodeFail)

	return nil
}

func ackCodeText(code uint32) string {
	switch code {
	case 0:
		return "成功"
	case 1:
		return "消息结构错"
	case 2:
		return "命令字错"
	case 3:
		return "序列号重复"
	case 4:
		return "消息长度错"
	case 5:
		return "资费代码错"
	case 6:
		return "超过最大长度"
	case 7:
		return "业务代码错"
	case 8:
		return "流量控制错"
	case 9:
		return "其他错误"
	default:
		return "未知"
	}
}
