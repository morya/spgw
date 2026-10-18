package app

import (
	"log"

	"github.com/nats-io/nats.go"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/morya/spgw/internal/global/consts"
)

var (
	BasePath  string          // 定义项目的根目录
	ConfigYml YmlConfigInterf // 全局配置文件指针

	EtcdClient *clientv3.Client
	NatsClient *nats.Conn

	GormDbMysql      *gorm.DB              // mysql数据库连接
	GormDbSqlserver  *gorm.DB              // sqlserver数据库连接
	GormDbPostgreSql *gorm.DB              // postgresql数据库连接
	ZapLog           *zap.Logger           // 全局日志指针
	CasbinV2         CasbinInterf          // casbin指针
	Cache            CacheInterf           // 缓存指针
	TokenService     TokenServiceInterface // token管理
	Response         ResponseHandler       // 全局响应指针
	UploadService    FileUploadService     // 文件上传服务
	JobScheduler     JobSchedulerInterf    // 全局任务调度器

	BillDir string // SP 话单文件根目录（portal 子命令 --bill-dir 设定，供话单下载使用）
)

/*
 * @Description: 获取数据库连接
 * @param sqlType 数据库类型 mysql sqlserver postgresql
 * @return *gorm.DB
 */
func DB(sqlType ...string) *gorm.DB {
	var dbType string
	if len(sqlType) > 0 {
		dbType = sqlType[0]
	} else {
		dbType = ConfigYml.GetString("gormv2.usedbtype")
	}
	var db *gorm.DB
	switch dbType {
	case consts.DbTypeMySql:
		db = GormDbMysql
	case consts.DbTypeSqlServer:
		db = GormDbSqlserver
	case consts.DbTypePostgreSql:
		db = GormDbPostgreSql
	default:
		db = GormDbMysql
	}
	if db == nil {
		log.Fatal("数据库连接失败")
	}

	return db
}
