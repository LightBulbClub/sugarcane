package model

import (
	"time"
)

// App 结构体持有所有核心依赖
type App struct {
	DataChannel  chan BandData
	AlertChannel chan Alert
}

// GlobalApp 应用程序的全局实例
var GlobalApp = App{
	DataChannel:  make(chan BandData, 1000),
	AlertChannel: make(chan Alert, 100),
}

// BandData 定义手环上传的数据结构
// 兼容 glucose 项目的两种格式：
//   - 格式1: {acceleration: (x,y,z), heart: (bpm,spo2), driverId}
//   - 格式2: {driver_id, heart_rate, accel_x, accel_y, accel_z}
type BandData struct {
	DriverID  string    `json:"driver_id"`
	Timestamp time.Time `json:"timestamp"`
	HeartRate int       `json:"heart_rate"`
	SpO2      float64   `json:"spo2,omitempty"`
	AccelX    float64   `json:"accel_x"`
	AccelY    float64   `json:"accel_y"`
	AccelZ    float64   `json:"accel_z"`
}

// GlucoseUpload 兼容 glucose MicroPython 项目的数据格式
type GlucoseUpload struct {
	Acceleration [3]float64 `json:"acceleration"`
	Heart        [2]float64 `json:"heart"` // [bpm, spo2]
	DriverID     string     `json:"driverId"`
}

// Alert 定义告警信息结构
type Alert struct {
	DriverID  string    `json:"driver_id"`
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
	Type      string    `json:"type"` // HeartRateAnomaly, SuddenMovement, etc.
}

// DrowsinessData 瞌睡检测数据结构 (来自 sucrose)
type DrowsinessData struct {
	DriverID   string    `json:"driver_id"`
	Timestamp  time.Time `json:"timestamp"`
	IsDrowsy   bool      `json:"is_drowsy"`
	EARValue   float64   `json:"ear_value"`
	Confidence float64   `json:"confidence"`
}

// DriverStatus 司机综合状态 (供 fructose 等客户端查询)
type DriverStatus struct {
	DriverID   string    `json:"driver_id"`
	HeartRate  int       `json:"heart_rate"`
	SpO2       float64   `json:"spo2"`
	IsDrowsy   bool      `json:"is_drowsy"`
	LastUpdate time.Time `json:"last_update"`
	Alerts     []string  `json:"alerts"`
}
