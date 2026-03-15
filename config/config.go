package config

import "time"

// InfluxDB配置
const (
	InfluxURL    = "http://localhost:8086" // 您的 InfluxDB 地址
	InfluxToken  = "your-influxdb-token"   // 您的 InfluxDB Token
	InfluxOrg    = "your-org"
	InfluxBucket = "driver_monitoring"
	Measurement  = "handband_data"
)

// Server配置
const (
	ListenPort = ":3000"
)

// Analysis配置
const (
	// 心率告警阈值
	HeartRateMin = 50
	HeartRateMax = 100
	// 加速度变化阈值 (例如，超过 2g 的突然变化)
	AccelThreshold = 20.0
	// 告警静默时间，避免重复告警
	AlertCooldown = 30 * time.Second
)
