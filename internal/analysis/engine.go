package analysis

import (
	"fmt"
	"log"
	"math"
	"sync"
	"time"

	"github.com/LightBulbClub/sugarcane/internal/config"
	"github.com/LightBulbClub/sugarcane/internal/model"
)

// AlertEngine 告警引擎
type AlertEngine struct {
	cooldownMap sync.Map // driver_id -> lastAlertTime
}

// NewAlertEngine 创建新的告警引擎
func NewAlertEngine() *AlertEngine {
	return &AlertEngine{}
}

// Start 启动实时分析和告警协程
func (e *AlertEngine) Start() {
	go e.processData()
	go e.processAlerts()
}

// processData 从 DataChannel 读取数据并执行分析
func (e *AlertEngine) processData() {
	for data := range model.GlobalApp.DataChannel {
		e.checkHeartRate(data)
		e.checkAcceleration(data)
		// TODO: 添加更多复杂的分析模型，如疲劳指数、微睡眠检测等
	}
}

// checkHeartRate 检测心率是否在正常区间
func (e *AlertEngine) checkHeartRate(d model.BandData) {
	if d.HeartRate < config.Cfg.Analysis.HeartRateMin || d.HeartRate > config.Cfg.Analysis.HeartRateMax {
		message := fmt.Sprintf("心率异常: %d bpm，超出正常范围 [%d, %d]",
			d.HeartRate, config.Cfg.Analysis.HeartRateMin, config.Cfg.Analysis.HeartRateMax)
		e.triggerAlert(d.DriverID, "HeartRateAnomaly", message)
	}
}

// checkAcceleration 检测加速度剧烈变化 (简易合向量检测)
func (e *AlertEngine) checkAcceleration(d model.BandData) {
	accelMagnitude := math.Sqrt(d.AccelX*d.AccelX + d.AccelY*d.AccelY + d.AccelZ*d.AccelZ)

	if accelMagnitude > config.Cfg.Analysis.AccelThreshold {
		message := fmt.Sprintf("加速度剧烈变化: %.2f (阈值 %.2f)，可能发生剧烈动作或碰撞。",
			accelMagnitude, config.Cfg.Analysis.AccelThreshold)
		e.triggerAlert(d.DriverID, "SuddenMovement", message)
	}
}

// triggerAlert 发送告警到 AlertChannel，并处理静默期
func (e *AlertEngine) triggerAlert(driverID, alertType, message string) {
	lastAlertTime, ok := e.cooldownMap.Load(driverID)
	if ok && time.Since(lastAlertTime.(time.Time)) < config.Cfg.Analysis.AlertCooldown {
		return
	}

	newAlert := model.Alert{
		DriverID:  driverID,
		Timestamp: time.Now(),
		Message:   message,
		Type:      alertType,
	}

	e.cooldownMap.Store(driverID, time.Now())
	model.GlobalApp.AlertChannel <- newAlert
}

// processAlerts 负责从 AlertChannel 中取出告警并执行通知
func (e *AlertEngine) processAlerts() {
	for alert := range model.GlobalApp.AlertChannel {
		// TODO: 实际的通知逻辑：发送邮件、短信、App 推送（例如通过 MQTT 或 WebHook）
		log.Printf("🚨 ALERT [%s] for Driver %s at %s: %s",
			alert.Type, alert.DriverID, alert.Timestamp.Format(time.RFC3339), alert.Message)
	}
}
