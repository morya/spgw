package gconfig

type DbSetting struct {
	User     string `yaml:"user" validate:"required"`
	Password string `yaml:"password" validate:"required"`
	Address  string `yaml:"address" validate:"required"`
	Sid      string `yaml:"sid" validate:"required"`
	Timezone string `yaml:"timezone" validate:"required"`
}
