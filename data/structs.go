package data

import (
	"time"

	"github.com/InfluxCommunity/influxdb3-go/v2/influxdb3"
)

// App 结构体持有所有核心依赖和通道
type App struct {
	InfluxClient *influxdb3.Client
	DataChannel  chan BandData
	AlertChannel chan Alert
}

// GlobalApp 应用程序的全局实例
var GlobalApp App

// BandData 定义手环上传的数据结构
type BandData struct {
	DriverID  string    `json:"driver_id"`
	Timestamp time.Time `json:"timestamp"`
	HeartRate int       `json:"heart_rate"`
	AccelX    float64   `json:"accel_x"`
	AccelY    float64   `json:"accel_y"`
	AccelZ    float64   `json:"accel_z"`
}

// Alert 定义告警信息结构
type Alert struct {
	DriverID  string
	Timestamp time.Time
	Message   string
	Type      string // HeartRateAnomaly, SuddenMovement, etc.
}

// SwitchChannel 全局通道，用于将数据从 HTTP 传递到分析引擎
var SwitchChannel = make(chan BandData, 1000)

// AlertChannel 全局通道，用于将告警信息从分析引擎传递到通知服务
var AlertChannel = make(chan Alert, 100)
