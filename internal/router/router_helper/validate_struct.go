package router_helper

import (
	stdjson "encoding/json"
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"

	"github.com/morya/spgw/internal/validatex"
)

type FillDefaultModel interface {
	FillDefaults()
}

func BindAndValid(c *gin.Context, i any) error {
	ignoreBindErr := c.Query("ignore-bind-error") != ""

	if err := jsoniter.NewDecoder(c.Request.Body).Decode(i); err != nil {
		// 保留原 echo 路径对 jsoniter 解码错误的更友好提示
		var ute *stdjson.UnmarshalTypeError
		var se *stdjson.SyntaxError
		if errors.As(err, &ute) {
			err = fmt.Errorf("Unmarshal type error: expected=%v, got=%v, field=%v, offset=%v",
				ute.Type, ute.Value, ute.Field, ute.Offset)
		} else if errors.As(err, &se) {
			err = fmt.Errorf("Syntax error: offset=%v, error=%v", se.Offset, se.Error())
		}
		if !ignoreBindErr {
			return err
		}
	}

	if p, ok := i.(FillDefaultModel); ok {
		// paginator 自动填充 defaults
		p.FillDefaults()
	}

	return validatex.Validate(i)
}
