package main

import (
	"os"
)

const (
	colorReset  = "\033[0m"
	colorBold   = "\033[1m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
)

// colorEnabled 控制是否输出 ANSI 颜色码。
// 仅当 stderr 是终端时启用，避免日志重定向到文件时混入控制字符。
var colorEnabled = isTerminal(os.Stderr)

func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// colorize 用指定颜色包裹字符串；非终端环境原样返回。
func colorize(s, color string) string {
	if !colorEnabled {
		return s
	}
	return color + s + colorReset
}

// colorizeBold 用指定颜色 + 加粗包裹字符串；非终端环境原样返回。
func colorizeBold(s, color string) string {
	if !colorEnabled {
		return s
	}
	return color + colorBold + s + colorReset
}

// colorizeStatusTag 根据 TestStatus 返回带颜色的标签字符串。
//   - PASS: 绿色加粗
//   - FAIL: 红色加粗
//   - SKIP: 黄色加粗
func colorizeStatusTag(status TestStatus) string {
	switch status {
	case StatusPass:
		return colorizeBold("[PASS]", colorGreen)
	case StatusFail:
		return colorizeBold("[FAIL]", colorRed)
	case StatusSkip:
		return colorizeBold("[SKIP]", colorYellow)
	default:
		return "[" + string(status) + "]"
	}
}
