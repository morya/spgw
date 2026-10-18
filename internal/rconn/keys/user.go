package keys

import (
	"fmt"
)

func GetRoleMenuKey(roleID uint) string {
	return fmt.Sprintf("role:menu:%d", roleID)
}

func GetApiMenuKey(apiID uint) string {
	return fmt.Sprintf("api:menu:%d", apiID)
}
