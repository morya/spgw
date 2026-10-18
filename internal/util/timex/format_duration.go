package timex

import (
	"fmt"
)

func HumanSeconds(total int64) string {
	var h, m, s int64

	h = total / 60 / 60
	m = total / 60 % 60
	s = total % 60

	return fmt.Sprintf("%v:%v:%v", h, m, s)
}
