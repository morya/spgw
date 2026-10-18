package models

import (
	"encoding/json"
	"os"
	"strings"
	"sync"

	"go.uber.org/zap"

	"github.com/morya/spgw/internal/global/app"
)

const AREAPATH = "./resource/public/area/area.json"

var (
	instance AreaModelList
	once     sync.Once
)

func NewAreaModel() *AreaModel {
	return &AreaModel{}
}

func GetAreaListInstance() AreaModelList {
	once.Do(func() {
		file, err := os.Open(AREAPATH)
		if err != nil {
			app.ZapLog.Error("打开地区数据文件失败",
				zap.String("path", AREAPATH),
				zap.Error(err))
			return
		}
		defer file.Close()

		if err := json.NewDecoder(file).Decode(&instance); err != nil {
			app.ZapLog.Error("解析地区数据文件失败",
				zap.String("path", AREAPATH),
				zap.Error(err))
		}
	})
	return instance
}

type AreaModel struct {
	Value    string        `json:"value"`
	Label    string        `json:"label"`
	Level    string        `json:"level"`
	Parent   string        `json:"parent"`
	Children AreaModelList `json:"children"`
}

type AreaModelList []AreaModel

func (list AreaModelList) IsEmpty() bool {
	return len(list) == 0
}

// AreaText 地区编码转换为地区文本
func (list AreaModelList) AreaText(area string, split string) string {
	if area == "" || list.IsEmpty() {
		return ""
	}
	areaList := strings.Split(area, ",")
	var areaText strings.Builder
	current := list
	for _, v := range areaList {
		for _, item := range current {
			if item.Value == v {
				areaText.WriteString(item.Label + split)
				current = item.Children
				break
			}
		}
	}
	return strings.TrimRight(areaText.String(), split)
}
