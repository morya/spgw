package keys

import (
	"fmt"
)

func SignOutKey(tokenID string) string {
	return fmt.Sprintf("sms-sys:logout_jwt:%v", tokenID)
}
