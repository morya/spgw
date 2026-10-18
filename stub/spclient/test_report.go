package main

import (
	"fmt"
	"os"
	"sort"
	"time"
)

func WriteTestResult(results []TestResult, resultFile string) error {
	f, err := os.Create(resultFile)
	if err != nil {
		return err
	}
	defer f.Close()

	w := func(format string, a ...any) {
		fmt.Fprintf(f, format, a...)
	}

	var pass, fail, skip int
	categories := make(map[string]struct{ pass, fail, skip int })
	for _, r := range results {
		switch r.Status {
		case StatusPass:
			pass++
		case StatusFail:
			fail++
		case StatusSkip:
			skip++
		}
		c := categories[r.Category]
		switch r.Status {
		case StatusPass:
			c.pass++
		case StatusFail:
			c.fail++
		case StatusSkip:
			c.skip++
		}
		categories[r.Category] = c
	}
	total := len(results)

	w("=== spgw 协议测试结果 ===\n")
	w("时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	w("总计: %d  PASS: %d  FAIL: %d  SKIP: %d\n\n", total, pass, fail, skip)

	w("--- 按分类统计 ---\n")
	cats := make([]string, 0, len(categories))
	for k := range categories {
		cats = append(cats, k)
	}
	sort.Strings(cats)
	for _, cat := range cats {
		c := categories[cat]
		w("  %-15s: total=%d  pass=%d  fail=%d  skip=%d\n", cat, c.pass+c.fail+c.skip, c.pass, c.fail, c.skip)
	}

	w("\n--- 详细结果 ---\n")
	w("%-25s %-30s %-8s %-12s %s\n", "用例ID", "名称", "状态", "耗时", "详情")
	w("%s\n", "-------------------------------------------------------------------------------------------------------------------------------")
	for _, r := range results {
		w("%-25s %-30s %-8s %-12s %s\n", r.ID, r.Name, r.Status, r.Duration.Round(time.Millisecond), r.Details)
		if r.Error != "" {
			w("  -> Error: %s\n", r.Error)
		}
	}

	w("\n--- 结论 ---\n")
	if fail == 0 {
		w("所有用例通过 (PASS=%d, SKIP=%d)\n", pass, skip)
	} else {
		w("有 %d 个用例失败\n", fail)
	}

	return nil
}
