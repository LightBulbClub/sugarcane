package handler

import (
	"log"
	"time"

	"github.com/LightBulbClub/sugarcane/config"
	"github.com/LightBulbClub/sugarcane/data"

	"github.com/gofiber/fiber/v2"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

// DataUploadHandler 处理手环上传数据的 POST 请求
func DataUploadHandler(c *fiber.Ctx) error {
	var uploadData data.HandbandData

	if err := c.BodyParser(&uploadData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid data format or request body",
		})
	}

	// 基本校验
	if uploadData.DriverID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "DriverID is required",
		})
	}

	// 始终使用服务器接收时间作为时间戳
	uploadData.Timestamp = time.Now()

	// --- 1. 写入 InfluxDB (持久化) ---
	writeToInfluxDB(uploadData)

	// --- 2. 传入分析通道 (异步分析) ---
	select {
	case data.GlobalApp.DataChannel <- uploadData:
		// 成功发送
	default:
		// 通道已满，防止阻塞
		log.Println("Warning: Data channel full, discarding data for", uploadData.DriverID)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Data received, persisted, and queued for analysis",
	})
}

// writeToInfluxDB 封装 InfluxDB 写入逻辑
func writeToInfluxDB(d data.HandbandData) {
	// 获取异步写入API
	writeAPI := data.GlobalApp.InfluxClient.WriteAPI(config.InfluxOrg, config.InfluxBucket)

	// 创建数据点 (Point)
	p := influxdb2.NewPointWithMeasurement(config.Measurement).
		AddTag("driver_id", d.DriverID).
		AddField("heart_rate", d.HeartRate).
		AddField("accel_x", d.AccelX).
		AddField("accel_y", d.AccelY).
		AddField("accel_z", d.AccelZ).
		SetTime(d.Timestamp)

	// 异步写入数据点
	writeAPI.WritePoint(p)

	// 生产环境中，您应该在单独的协程中监听 writeAPI.Errors()
	// 检查错误，确保数据最终写入成功。
}
