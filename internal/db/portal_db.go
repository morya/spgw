package db

import (
	"errors"
	"log"

	"github.com/morya/spgw/internal/global/app"
	"github.com/morya/spgw/internal/global/myerrors"
	"github.com/morya/spgw/internal/utils/gormhelper"
)

func InitDB() error {
	// mysql
	if app.ConfigYml.GetInt("gormv2.mysql.isinitglobalgormmysql") == 1 {
		if dbMysql, err := gormhelper.GetOneMysqlClient(); err != nil {
			log.Fatal(myerrors.ErrorsGormInitFail + err.Error())
			return err
		} else {
			app.GormDbMysql = dbMysql
		}
	} else {
		return errors.New("only mysql support")
	}

	return nil
}
