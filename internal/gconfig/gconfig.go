package gconfig

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/morya/gommon/config"

	"github.com/morya/spgw/internal/validatex"
)

var Cfg = &GConfig{}

// GConfig global config
type GConfig struct {
	ProdRun  bool
	SqlDebug bool

	Agent struct {
		MaxConcurrency int `yaml:"max_concurrency" validate:"required,min=1000,max=10000"` // 单节点最大限频
		Burst          int `yaml:"burst" validate:"required,min=1,max=5000"`               // 突发数(建议1000)

		HeartbeatInterval int `yaml:"heartbeat_interval" validate:"required,min=3,max=60"`

		// LinkQueueCap 单条 Link 发送队列容量，满则 SendMsg 返回 link.ErrQueueFull。0 兜底默认值。
		LinkQueueCap int `yaml:"link_queue_cap" validate:"omitempty,min=16,max=65536"`
		// AppQueueCap iagent/oagent App 主队列容量。0 兜底默认值。
		AppQueueCap int `yaml:"app_queue_cap" validate:"omitempty,min=256,max=1048576"`
		// WriteTimeout Conn.Write 单次写超时。0 兜底 1s。
		WriteTimeout config.Duration `yaml:"write_timeout" validate:"omitempty"`
	} `yaml:"agent"`

	Server struct {
		SPGwId    int `yaml:"sp_gw_id" validate:"required,min=1,max=2097152"` // 网关ID 21个bit位可用范围
		RouteMode int `yaml:"route_mode" validate:"required,min=1,max=2"`     // 1:按比例路由 2:按归属地路由

		WaitAckTimeout config.Duration `yaml:"wait_ack_timeout" validate:"required"` // 等待确认超时时间
		WaitRptTimeout config.Duration `yaml:"wait_rpt_timeout" validate:"required"` // 等待状态报告超时时间

		MaxWaitAck int `yaml:"max_wait_ack" validate:"required,min=1,max=1000000"` // 最大待确认数
		MaxWaitRpt int `yaml:"max_wait_rpt" validate:"required,min=1,max=1000000"` // 最大待状态报告数

		MaxConcurrency int `yaml:"max_concurrency" validate:"required,min=1000,max=10000"` // 单节点最大限频
		Burst          int `yaml:"burst" validate:"required,min=1,max=5000"`               // 突发数(建议1000)

		// AppQueueCap server App 主队列容量。0 兜底默认值。
		AppQueueCap int `yaml:"app_queue_cap" validate:"omitempty,min=256,max=1048576"`

		StatFlushInterval config.Duration `yaml:"stat_flush_interval" validate:"required"` // 统计计数落库间隔（范围在 Validate() 中校验）
	} `yaml:"server"`

	RedisAddress string            `yaml:"redis_address" validate:"required"`
	Etcd         EtcdSetting       `yaml:"etcd" validate:"required"`
	MainDB       DbSetting         `yaml:"db" validate:"required"`
	Nats         NatsSetting       `yaml:"nats" validate:"required"`
	BillDumper   BillDumperSetting `yaml:"billdumper" validate:"required"`
}

type EtcdSetting struct {
	Endpoints        []string        `yaml:"endpoints" validate:"required"`
	AutoSyncInterval config.Duration `yaml:"auto_sync_interval" validate:"required"`
}

type NatsSetting struct {
	Address       string          `yaml:"address" validate:"required"`
	MaxReconnects int             `yaml:"max_reconnects" validate:"min=-1"` // -1=无限重连
	ReconnectWait config.Duration `yaml:"reconnect_wait" validate:"required"`
	Timeout       config.Duration `yaml:"timeout" validate:"required"`
}

// BillDumperSetting 话单写出配置（cmd/billdumper 进程使用）
type BillDumperSetting struct {
	OutputPath    string `yaml:"output_path" validate:"required,min=1,max=4096"`       // 输出根目录，全局与账号文件均写入此目录
	MaxFileSizeMB int    `yaml:"max_file_size_mb" validate:"required,min=1,max=10240"` // 单文件大小上限（MB），lumberjack 切割
	MaxBackups    int    `yaml:"max_backups" validate:"omitempty,min=1,max=10000"`     // 保留旧文件份数，0 兜底 30
	MaxAgeDays    int    `yaml:"max_age_days" validate:"omitempty,min=1,max=3650"`     // 按天保留，0=不按天删
	Compress      bool   `yaml:"compress"`

	SpRotateIntervalSec int `yaml:"sp_rotate_interval_sec" validate:"required,min=60,max=3600"` // 账号文件按时间切分间隔（秒），切分后归档 DB
}

func (c *GConfig) Validate() error {
	if err := validatex.Validate(c); err != nil {
		return err
	}

	// Duration 范围校验（config.Duration 是 struct 类型，validate tag 不支持 min/max）
	if c.Server.WaitAckTimeout.Duration < 1*time.Second || c.Server.WaitAckTimeout.Duration > 30*time.Minute {
		return fmt.Errorf("wait_ack_timeout must be between 1s and 30m, got %v", c.Server.WaitAckTimeout)
	}
	if c.Server.WaitRptTimeout.Duration < 1*time.Second || c.Server.WaitRptTimeout.Duration > 24*time.Hour {
		return fmt.Errorf("wait_rpt_timeout must be between 1s and 24h, got %v", c.Server.WaitRptTimeout)
	}
	if c.Server.StatFlushInterval.Duration < 5*time.Second || c.Server.StatFlushInterval.Duration > 1*time.Hour {
		return fmt.Errorf("stat_flush_interval must be between 5s and 1h, got %v", c.Server.StatFlushInterval)
	}

	// Agent.WriteTimeout 范围校验；0 表示用代码兜底默认值 1s，不强制配置
	if c.Agent.WriteTimeout.Duration > 0 {
		if c.Agent.WriteTimeout.Duration < 100*time.Millisecond || c.Agent.WriteTimeout.Duration > 30*time.Second {
			return fmt.Errorf("agent.write_timeout must be between 100ms and 30s, got %v", c.Agent.WriteTimeout)
		}
	}

	return nil
}

// LinkQueueCap 兜底返回 Link 队列容量，v < 16 时返回 1024
func LinkQueueCap(v int) int {
	if v >= 16 {
		return v
	}
	return 1024
}

// AppQueueCap 兜底返回 iagent/oagent App 队列容量，v < 256 时返回 1024
func AppQueueCap(v int) int {
	if v >= 256 {
		return v
	}
	return 1024
}

// ServerAppQueueCap 兜底返回 server App 队列容量，v < 256 时返回 10240
func ServerAppQueueCap(v int) int {
	if v >= 256 {
		return v
	}
	return 10240
}

// WriteTimeout 兜底返回 Conn.Write 超时，d <= 0 时返回 1s
func WriteTimeout(d time.Duration) time.Duration {
	if d > 0 {
		return d
	}
	return time.Second
}

func Load(fp string) error {
	f, err := os.Open(fp)
	if err != nil {
		return fmt.Errorf("open config file %s: %w", fp, err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return fmt.Errorf("read config file %s: %w", fp, err)
	}

	dec := yaml.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(Cfg); err != nil {
		return fmt.Errorf("decode config file %s: %w", fp, err)
	}
	return Cfg.Validate()
}

func WatchChange() error {
	return nil
}
