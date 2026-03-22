package handler

import (
	"context"
	"log"
	"time"

	"github.com/LightBulbClub/sugarcane/config"
	"github.com/LightBulbClub/sugarcane/data"

	"github.com/InfluxCommunity/influxdb3-go/v2/influxdb3"
	"github.com/gofiber/fiber/v2"
)

// DataUploadHandler 处理手环上传数据的 POST 请求
func DataUploadHandler(c *fiber.Ctx) error {
	var uploadData data.BandData

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
func writeToInfluxDB(d data.BandData) {
	// 获取异步写入API
	client := data.GlobalApp.InfluxClient

	// 创建数据点 (Point)
	p := influxdb3.NewPointWithMeasurement(config.Cfg.Influx.Measurement).
		SetTag("driver_id", d.DriverID).
		SetField("heart_rate", d.HeartRate).
		SetField("accel_x", d.AccelX).
		SetField("accel_y", d.AccelY).
		SetField("accel_z", d.AccelZ).
		SetTimestamp(d.Timestamp)

	points := []*influxdb3.Point{p}

	// 异步写入数据点
	err := client.WritePoints(context.Background(), points)
	if err != nil {
		log.Printf("Error writing to InfluxDB for driver %s: %v", d.DriverID, err)
	}
}
