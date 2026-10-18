package db

import (
	"sync"
	"time"

	mysql1 "github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
	"gorm.io/gorm/schema"

	config "github.com/morya/spgw/internal/gconfig"
	"github.com/morya/spgw/internal/glog"
)

var (
	MainDB    *gorm.DB
	onceClose = &sync.Once{}
	MyNaming  = schema.NamingStrategy{
		TablePrefix:   "gw_",
		SingularTable: true,
	}
)

func BuildMsqlConfig(setting config.DbSetting) (cfg mysql.Config, err error) {
	c := mysql1.NewConfig()
	c.User = setting.User
	c.Passwd = setting.Password
	c.Addr = setting.Address
	c.DBName = setting.Sid
	c.Net = "tcp"

	c.ParseTime = true
	c.AllowNativePasswords = true
	c.AllowOldPasswords = true
	c.Timeout = time.Second * 5 // Dial timeout
	c.ReadTimeout = 30 * time.Second
	c.WriteTimeout = 30 * time.Second
	c.Params = map[string]string{
		"charset": "utf8mb4",
	}

	if setting.Timezone != "" {
		var loc *time.Location
		loc, err = time.LoadLocation(setting.Timezone)
		if err != nil {
			err = errors.Wrap(err, "parse timezone failed")
			return
		}
		c.Loc = loc
	}

	dsn := c.FormatDSN()
	cfg.DSN = dsn

	return
}

func ConnectDB(setting config.DbSetting) (d *gorm.DB, err error) {
	gormCfg := &gorm.Config{
		NamingStrategy:                           MyNaming,
		DisableForeignKeyConstraintWhenMigrating: true,
	}

	// disable log color
	if config.Cfg.ProdRun {
		gormCfg.Logger = glogger.New(glog.GormLogger, glogger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  glogger.Warn,
			IgnoreRecordNotFoundError: false,
			Colorful:                  false,
		})
	}

	cfg, err := BuildMsqlConfig(setting)
	if err != nil {
		return
	}

	d, err = gorm.Open(mysql.New(cfg), gormCfg)
	if err != nil {
		return nil, errors.Wrap(err, "database connect failed")
	}

	db, _ := d.DB()
	db.SetMaxIdleConns(5)
	db.SetMaxOpenConns(50)
	db.SetConnMaxLifetime(time.Minute * 5)
	db.SetConnMaxIdleTime(time.Minute * 5)
	if config.Cfg.SqlDebug {
		d = d.Debug()
	}

	return d, nil
}

func ConnectMainDB(s config.DbSetting) (err error) {
	MainDB, err = ConnectDB(s)
	if err != nil {
		zap.S().With(zap.Any("setting", s)).With(zap.Any("err", err)).Errorf("database connect failed")
		return err
	}
	zap.S().With(
		zap.String("user", s.User),
		zap.Any("host", s.Address),
		zap.Any("sid", s.Sid),
		zap.Any("timezone", s.Timezone),
	).Debug("database connected")
	return nil
}

func Close() error {
	onceClose.Do(func() {
		if MainDB != nil {
			d, _ := MainDB.DB()
			d.Close()
		}
		MainDB = nil
	})
	return nil
}
