package models

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"slices"
)

type StrList []string

func (s *StrList) Scan(value any) error {
	data, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal JSONB value:%v", value)
	}

	data = []byte(bytes.TrimSpace(data))
	if len(data) == 0 {
		return nil
	}

	err := json.Unmarshal(data, s)
	return err
}

func (s StrList) Value() (driver.Value, error) {
	if len(s) == 0 {
		return nil, nil
	}
	return json.Marshal(s)
}

func (s StrList) Contains(str string) bool {
	return slices.Contains(s, str)
}
