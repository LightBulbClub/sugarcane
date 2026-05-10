package config

import (
	"os"
	"time"

	"github.com/BurntSushi/toml"
)

// InfluxConfig 配置结构体
type InfluxConfig struct {
	URL         string `toml:"url"`
	Token       string `toml:"token"`
	Org         string `toml:"org"`
	Bucket      string `toml:"bucket"`
	Measurement string `toml:"measurement"`
	Database    string `toml:"database"`
}

type ServerConfig struct {
	ListenPort string `toml:"listen_port"`
	APIToken   string `toml:"api_token"`
}

type AnalysisConfig struct {
	HeartRateMin   int           `toml:"heart_rate_min"`
	HeartRateMax   int           `toml:"heart_rate_max"`
	AccelThreshold float64       `toml:"accel_threshold"`
	AlertCooldown  time.Duration `toml:"alert_cooldown"`
}

type Config struct {
	Influx   InfluxConfig   `toml:"influx"`
	Server   ServerConfig   `toml:"server"`
	Analysis AnalysisConfig `toml:"analysis"`
}

// Cfg 包级变量，包含最终使用的配置（会有默认值，Load 会覆盖提供的字段）
var Cfg = &Config{
	Influx: InfluxConfig{
		URL:         "http://localhost:8181",
		Token:       "",
		Bucket:      "driver_monitoring",
		Database:    "sugarcane",
		Measurement: "band_data",
	},
	Server: ServerConfig{
		ListenPort: ":3000",
	},
	Analysis: AnalysisConfig{
		HeartRateMin:   50,
		HeartRateMax:   100,
		AccelThreshold: 2.0,
		AlertCooldown:  30 * time.Second,
	},
}

// Load 从给定路径加载 TOML 配置文件（如果 path 为空则使用 config.toml）。
// 如果文件不存在，则保留默认值并返回 nil。
func Load(path string) error {
	if path == "" {
		path = "config.toml"
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// 配置文件不存在：保留默认配置
		return nil
	}

	var fileCfg Config
	if _, err := toml.DecodeFile(path, &fileCfg); err != nil {
		return err
	}

	// 仅在 TOML 提供值时覆盖默认值
	if fileCfg.Influx.URL != "" {
		Cfg.Influx.URL = fileCfg.Influx.URL
	}
	if fileCfg.Influx.Token != "" {
		Cfg.Influx.Token = fileCfg.Influx.Token
	}
	if fileCfg.Influx.Org != "" {
		Cfg.Influx.Org = fileCfg.Influx.Org
	}
	if fileCfg.Influx.Bucket != "" {
		Cfg.Influx.Bucket = fileCfg.Influx.Bucket
	}
	if fileCfg.Influx.Measurement != "" {
		Cfg.Influx.Measurement = fileCfg.Influx.Measurement
	}
	if fileCfg.Influx.Database != "" {
		Cfg.Influx.Database = fileCfg.Influx.Database
	}
	if fileCfg.Server.ListenPort != "" {
		Cfg.Server.ListenPort = fileCfg.Server.ListenPort
	}
	if fileCfg.Server.APIToken != "" {
		Cfg.Server.APIToken = fileCfg.Server.APIToken
	}
	if fileCfg.Analysis.HeartRateMin != 0 {
		Cfg.Analysis.HeartRateMin = fileCfg.Analysis.HeartRateMin
	}
	if fileCfg.Analysis.HeartRateMax != 0 {
		Cfg.Analysis.HeartRateMax = fileCfg.Analysis.HeartRateMax
	}
	if fileCfg.Analysis.AccelThreshold != 0 {
		Cfg.Analysis.AccelThreshold = fileCfg.Analysis.AccelThreshold
	}

	return nil
}
