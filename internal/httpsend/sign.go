package httpsend

import (
	"fmt"
	"sort"
	"strings"

	"github.com/morya/gommon/strmd5"
)

// SignMD5 通用的 MD5 签名生成
// 参数按 ASCII 升序，排除 sign 和 phones 字段，空值也排除
// 结果转大写
func SignMD5(params map[string]string, apiKey string) string {
	keys := make([]string, 0, len(params))
	for k, v := range params {
		if k == "sign" || k == "key" {
			continue
		}
		if k == "phones" {
			continue
		}
		if v == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	for _, k := range keys {
		sb.WriteString(k)
		sb.WriteString("=")
		sb.WriteString(params[k])
		sb.WriteString("&")
	}
	sb.WriteString("key=")
	sb.WriteString(apiKey)

	raw := sb.String()
	md5Str := strmd5.StrMd5(raw)
	return strings.ToUpper(md5Str)
}

// VerifySign 验证签名
func VerifySign(params map[string]string, apiKey string, expectedSign string) error {
	calculated := SignMD5(params, apiKey)
	if calculated != expectedSign {
		return fmt.Errorf("sign mismatch: calculated=%s, expected=%s", calculated, expectedSign)
	}
	return nil
}
