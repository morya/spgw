package jsonx

import (
	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"github.com/json-iterator/go/extra"
	"github.com/wI2L/jettison"
)

func init() {
	// 前端给的字符串包裹的数据等不标准格式，能够直接兼容 主要为兼容php
	extra.RegisterFuzzyDecoders()
}

// DecodeRequest 使用 jsoniter（已注册 fuzzy decoders）解码请求体到 i。
func DecodeRequest(c *gin.Context, i any) error {
	return jsoniter.NewDecoder(c.Request.Body).Decode(i)
}

// WriteJSON 使用 jettison 编码（NilSliceEmpty / NilMapEmpty）并写回响应。
// 保持原 echo MyJSONSerializer 输出行为：nil slice -> [], nil map -> {}。
func WriteJSON(c *gin.Context, status int, v any) error {
	data, err := jettison.MarshalOpts(v, jettison.NilSliceEmpty(), jettison.NilMapEmpty())
	if err != nil {
		return err
	}
	c.Data(status, "application/json", data)
	return nil
}
