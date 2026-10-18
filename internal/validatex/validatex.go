package validatex

import (
	"errors"
	"reflect"
	"strings"

	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	zh_translations "github.com/go-playground/validator/v10/translations/zh"
)

var (
	v     *validator.Validate
	trans ut.Translator
)

func init() {
	v = validator.New()

	zhLocale := zh.New()
	uni := ut.New(zhLocale, zhLocale)
	trans, _ = uni.GetTranslator("zh")
	_ = zh_translations.RegisterDefaultTranslations(v, trans)

	// 用 json/form tag 名作为错误消息中的字段名
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" || name == "" {
			name = strings.SplitN(fld.Tag.Get("form"), ",", 2)[0]
		}
		if name == "-" || name == "" {
			return fld.Name
		}
		return name
	})
}

// Validate 校验 struct，返回第一个错误的中文翻译。
func Validate(s any) error {
	err := v.Struct(s)
	if err == nil {
		return nil
	}
	var errs validator.ValidationErrors
	if !errors.As(err, &errs) || len(errs) == 0 {
		return err
	}
	return errors.New(errs[0].Translate(trans))
}

// Engine 返回底层 validator 实例，供 Gin binding 复用。
func Engine() *validator.Validate {
	return v
}

// Translator 返回中文翻译器实例。
func Translator() ut.Translator {
	return trans
}
