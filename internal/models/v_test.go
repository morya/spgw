package models

import (
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
)

// TestValidateLinkInfo 覆盖 LinkInfo 上各 validate tag 的边界情况。
// 字段规则：
//   - OAccountID: required
//   - Spid:        max=6,alphanum
//   - Pswd:        max=16,alphanum
//   - RemoteAddress: hostname_port
//   - LocalAddress:  max=128
//   - Proto:         max=32,oneof=cmpp30 cmpp20
//
// 合法基线：Spid/Pswd 填 alphanum 值，Proto=cmpp30，OAccountID 非零，
// 仅在被测字段上变异，便于定位失败原因。
func TestValidateLinkInfo(t *testing.T) {
	base := func() LinkInfo {
		return LinkInfo{
			IAccountID:    1,
			OAccountID:    2,
			Spid:          "spid01",
			Pswd:          "pass1234",
			Proto:         "cmpp30",
			RemoteAddress: "127.0.0.1:8888",
		}
	}

	tests := []struct {
		name    string
		mutate  func(*LinkInfo)
		wantErr bool
		// errSubstr 期望错误信息包含的子串（wantErr=true 时校验）
		errSubstr string
	}{
		// ---- 合法用例 ----
		{"valid ipv4:port", func(l *LinkInfo) {}, false, ""},
		{"valid hostname:port", func(l *LinkInfo) {
			l.RemoteAddress = "spgw.example.com:7890"
		}, false, ""},

		// ---- RemoteAddress 边界 ----
		{"remote missing port", func(l *LinkInfo) {
			l.RemoteAddress = "127.0.0.1"
		}, true, "RemoteAddress"},
		{"remote empty", func(l *LinkInfo) {
			l.RemoteAddress = ""
		}, true, "RemoteAddress"},

		// ---- Spid / Pswd ----
		{"spid empty fails alphanum", func(l *LinkInfo) {
			l.Spid = ""
		}, true, "Spid"},
		{"spid non-alphanum", func(l *LinkInfo) {
			l.Spid = "sp@id"
		}, true, "Spid"},
		{"spid too long", func(l *LinkInfo) {
			l.Spid = "abcdefg" // max=6
		}, true, "Spid"},
		{"pswd empty fails alphanum", func(l *LinkInfo) {
			l.Pswd = ""
		}, true, "Pswd"},

		// ---- Proto ----
		{"proto invalid value", func(l *LinkInfo) {
			l.Proto = "smpp"
		}, true, "Proto"},
		{"proto empty fails oneof", func(l *LinkInfo) {
			l.Proto = ""
		}, true, "Proto"},

		// ---- OAccountID ----
		{"oaccount id zero fails required", func(l *LinkInfo) {
			l.OAccountID = 0
		}, true, "OAccountID"},
	}

	v := validator.New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := base()
			tt.mutate(&item)
			err := v.Struct(item)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected validation error, got nil. item=%+v", item)
				}
				if tt.errSubstr != "" && !strings.Contains(err.Error(), tt.errSubstr) {
					t.Fatalf("expected error containing %q, got: %v", tt.errSubstr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected validation to pass, got error: %v. item=%+v", err, item)
			}
		})
	}
}
