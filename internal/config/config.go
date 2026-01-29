package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	App          AppConfig          `mapstructure:"app"`
	Server       ServerConfig       `mapstructure:"server"`
	Redis        RedisConfig        `mapstructure:"redis"`
	Queues       QueuesConfig       `mapstructure:"queues"`
	Logging      LoggingConfig      `mapstructure:"logging"`
	Progress     ProgressConfig     `mapstructure:"progress"`
	GRPCServices GRPCServicesConfig `mapstructure:"grpc_services"`
}

type AppConfig struct {
	Name string `mapstructure:"name"`
	Env  string `mapstructure:"env"`
}

type ServerConfig struct {
	HTTP   HTTPConfig   `mapstructure:"http"`
	Worker WorkerConfig `mapstructure:"worker"`
}

type HTTPConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type WorkerConfig struct {
	Concurrency int                `mapstructure:"concurrency"`
	Health      WorkerHealthConfig `mapstructure:"health"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type QueuesConfig struct {
	Critical int `mapstructure:"critical"`
	High     int `mapstructure:"high"`
	Default  int `mapstructure:"default"`
	Low      int `mapstructure:"low"`
}

type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

type ProgressConfig struct {
	MaxLen      int64         `mapstructure:"max_len"`
	TTL         time.Duration `mapstructure:"ttl"`
	ReadTimeout time.Duration `mapstructure:"read_timeout"`
}

type WorkerHealthConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Host    string `mapstructure:"host"`
	Port    int    `mapstructure:"port"`
}

// GRPCServicesConfig gRPC 服务配置
type GRPCServicesConfig struct {
	// Enabled 是否启用 gRPC 服务集成
	Enabled bool `mapstructure:"enabled"`
	// Services 服务注册表
	Services map[string]GRPCServiceConfig `mapstructure:"services"`
	// Defaults 默认配置
	Defaults GRPCServiceConfig `mapstructure:"defaults"`
}

// GRPCServiceConfig 单个 gRPC 服务配置
type GRPCServiceConfig struct {
	// Address 服务地址
	Address string `mapstructure:"address"`
	// Timeout 超时时间
	Timeout time.Duration `mapstructure:"timeout"`
	// HealthCheckInterval 健康检查间隔
	HealthCheckInterval time.Duration `mapstructure:"health_check_interval"`
	// MaxRetries 最大重试次数
	MaxRetries int `mapstructure:"max_retries"`
	// RetryDelay 重试延迟
	RetryDelay time.Duration `mapstructure:"retry_delay"`
}

func Load(configPath string) (*Config, error) {
	v := viper.New()

	v.SetConfigType("yaml")

	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.AddConfigPath("./configs")
		v.AddConfigPath(".")
	}

	v.SetEnvPrefix("TASKFLOW")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	cfg.applyDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) applyDefaults() {
	if c.Progress.MaxLen == 0 {
		c.Progress.MaxLen = 1000
	}
	if c.Progress.TTL == 0 {
		c.Progress.TTL = time.Hour
	}
	if c.Progress.ReadTimeout == 0 {
		c.Progress.ReadTimeout = 30 * time.Second
	}
}

func (c *Config) Validate() error {
	if c.Server.HTTP.Port <= 0 {
		return fmt.Errorf("server.http.port must be greater than 0")
	}
	if c.Server.Worker.Concurrency <= 0 {
		return fmt.Errorf("server.worker.concurrency must be greater than 0")
	}
	if c.Queues.Critical <= 0 || c.Queues.High <= 0 || c.Queues.Default <= 0 || c.Queues.Low <= 0 {
		return fmt.Errorf("queues weights must be greater than 0")
	}
	if c.Progress.MaxLen < 0 {
		return fmt.Errorf("progress.max_len must be greater than or equal to 0")
	}
	if c.Progress.TTL < 0 {
		return fmt.Errorf("progress.ttl must be greater than or equal to 0")
	}
	if c.Progress.ReadTimeout < 0 {
		return fmt.Errorf("progress.read_timeout must be greater than or equal to 0")
	}
	if c.Server.Worker.Health.Enabled {
		if c.Server.Worker.Health.Port <= 0 {
			return fmt.Errorf("server.worker.health.port must be greater than 0")
		}
	}
	return nil
}

func (c *Config) IsDevelopment() bool {
	return c.App.Env == "development"
}

func (c *Config) IsProduction() bool {
	return c.App.Env == "production"
}

func (c *QueuesConfig) ToMap() map[string]int {
	return map[string]int{
		"critical": c.Critical,
		"high":     c.High,
		"default":  c.Default,
		"low":      c.Low,
	}
}
