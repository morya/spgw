package migrate

import (
	"fmt"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/morya/spgw/internal/db"
	"github.com/morya/spgw/internal/models"
)

func TableName(d any) string {
	stmt := &gorm.Statement{DB: db.MainDB}
	if err := stmt.Parse(d); err != nil {
		return ""
	}
	return stmt.Schema.Table // 计算得出数据库表名称
}

func MigrateDB() error {
	if db.MainDB == nil {
		return fmt.Errorf("main db is nil")
	}
	m := db.MainDB.Migrator()
	arr := []any{
		// portal 管理后台表
		new(models.User),
		new(models.SysRole),
		new(models.SysMenu),
		new(models.SysApi),
		new(models.SysMenuApi),
		new(models.SysRoleMenu),
		new(models.SysUserRole),
		new(models.Tenant),
		new(models.SysUserTenant),
		new(models.SysDepartment),
		new(models.SysDict),
		new(models.SysDictItem),
		new(models.SysOperationLog),
		new(models.SysAffix),
		new(models.SysAffixChunk),
		new(models.SysGen),
		new(models.SysGenField),
		new(models.SysJobs),
		new(models.SysJobResults),

		// spgw 业务表
		new(models.SpInfo),

		new(models.OAccount),
		new(models.LinkInfo),
		new(models.Route),
		new(models.IStat),
		new(models.OStat),

		new(models.SpWallet),
		new(models.SpWalletLog),

		new(models.SpBillFile),
	}
	for _, item := range arr {
		if err := m.AutoMigrate(item); err != nil {
			table := TableName(item)

			zap.S().With(
				zap.String("table", table),
				zap.Error(err),
			).Errorf("auto migrate table %s failed", table)
			// return fmt.Errorf("auto migrate %T failed: %w", item, err)
		}
	}
	return nil
}
