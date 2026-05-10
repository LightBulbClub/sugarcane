package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/LightBulbClub/sugarcane/internal/config"
	"github.com/LightBulbClub/sugarcane/internal/model"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

// HTTPServer 封装 HTTP 服务器
type HTTPServer struct {
	app          *fiber.App
	influxService *InfluxDBService
}

// NewHTTPServer 创建新的 HTTP 服务器
func NewHTTPServer(influxService *InfluxDBService) *HTTPServer {
	app := fiber.New(fiber.Config{
		Prefork: false,
	})

	s := &HTTPServer{
		app:          app,
		influxService: influxService,
	}

	// 添加中间件
	app.Use(logger.New())
	app.Use(cors.New())

	// 健康检查（无需鉴权）
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
			"time":   "now",
		})
	})

	// 需要鉴权的路由组
	api := app.Group("/data", s.authMiddleware())
	api.Post("/upload", s.handleUpload)
	api.Post("/drowsiness", s.handleDrowsiness)
	api.Get("/driver/:id/status", s.handleDriverStatus)

	return s
}

// Start 启动 HTTP 服务器（阻塞）
func (s *HTTPServer) Start() {
	log.Printf("Fiber server starting on %s...", config.Cfg.Server.ListenPort)
	log.Fatal(s.app.Listen(config.Cfg.Server.ListenPort))
}

// handleUpload 处理手环上传数据的 POST 请求
// 兼容 glucose MicroPython 项目的数据格式和标准格式
func (s *HTTPServer) handleUpload(c *fiber.Ctx) error {
	body := c.Body()

	// 尝试解析为 glucose 格式
	var glucoseData model.GlucoseUpload
	if err := json.Unmarshal(body, &glucoseData); err == nil && glucoseData.DriverID != "" && len(glucoseData.Acceleration) == 3 {
		return s.processGlucoseData(c, glucoseData)
	}

	// 尝试解析为标准格式
	var standardData model.BandData
	if err := json.Unmarshal(body, &standardData); err == nil && standardData.DriverID != "" {
		return s.processStandardData(c, standardData)
	}

	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
		"error": "Invalid data format. Expected: {driver_id, heart_rate, accel_x, accel_y, accel_z} or {acceleration, heart, driverId}",
	})
}

// processGlucoseData 处理 glucose 格式的数据
func (s *HTTPServer) processGlucoseData(c *fiber.Ctx, glucoseData model.GlucoseUpload) error {
	// 转换为标准格式
	bandData := model.BandData{
		DriverID:  glucoseData.DriverID,
		Timestamp: time.Now(),
		HeartRate: int(glucoseData.Heart[0]),
		SpO2:      glucoseData.Heart[1],
		AccelX:    glucoseData.Acceleration[0],
		AccelY:    glucoseData.Acceleration[1],
		AccelZ:    glucoseData.Acceleration[2],
	}

	return s.processAndRespond(c, bandData)
}

// processStandardData 处理标准格式的数据
func (s *HTTPServer) processStandardData(c *fiber.Ctx, standardData model.BandData) error {
	standardData.Timestamp = time.Now()
	return s.processAndRespond(c, standardData)
}

// processAndRespond 处理数据并返回响应
func (s *HTTPServer) processAndRespond(c *fiber.Ctx, data model.BandData) error {
	// 1. 写入 InfluxDB
	if err := s.influxService.WriteBandData(context.Background(), data); err != nil {
		log.Printf("Error writing to InfluxDB for driver %s: %v", data.DriverID, err)
		// 继续处理，不阻塞响应
	}

	// 2. 传入分析通道
	select {
	case model.GlobalApp.DataChannel <- data:
		// 成功发送
	default:
		log.Printf("Warning: Data channel full, discarding data for %s", data.DriverID)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":    "Data received successfully",
		"driver_id":  data.DriverID,
		"heart_rate": data.HeartRate,
	})
}

// handleDrowsiness 处理瞌睡检测数据上传
func (s *HTTPServer) handleDrowsiness(c *fiber.Ctx) error {
	var drowsinessData model.DrowsinessData
	if err := c.BodyParser(&drowsinessData); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid JSON format",
		})
	}

	if drowsinessData.DriverID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "driver_id is required",
		})
	}

	drowsinessData.Timestamp = time.Now()

	// 写入 InfluxDB
	if err := s.influxService.WriteDrowsinessData(context.Background(), drowsinessData); err != nil {
		log.Printf("Error writing drowsiness data to InfluxDB for driver %s: %v", drowsinessData.DriverID, err)
	}

	// 如果检测到瞌睡，触发告警
	if drowsinessData.IsDrowsy {
		alert := model.Alert{
			DriverID:  drowsinessData.DriverID,
			Timestamp: drowsinessData.Timestamp,
			Message:   fmt.Sprintf("检测到司机瞌睡! EAR=%.3f, 置信度=%.2f%%", drowsinessData.EARValue, drowsinessData.Confidence*100),
			Type:      "DrowsinessDetected",
		}
		select {
		case model.GlobalApp.AlertChannel <- alert:
		default:
			log.Printf("Warning: Alert channel full, discarding drowsiness alert for %s", drowsinessData.DriverID)
		}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message":   "Drowsiness data received successfully",
		"driver_id": drowsinessData.DriverID,
		"is_drowsy": drowsinessData.IsDrowsy,
	})
}

// handleDriverStatus 获取司机综合状态
func (s *HTTPServer) handleDriverStatus(c *fiber.Ctx) error {
	driverID := c.Params("id")
	if driverID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "driver id is required",
		})
	}

	status, err := s.influxService.GetDriverStatus(context.Background(), driverID)
	if err != nil {
		log.Printf("Error getting driver status for %s: %v", driverID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get driver status",
		})
	}

	return c.JSON(status)
}

// authMiddleware 鉴权中间件，验证 Bearer Token
func (s *HTTPServer) authMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// 如果未配置 token，则跳过鉴权
		if config.Cfg.Server.APIToken == "" {
			return c.Next()
		}

		// 从 Authorization header 获取 token
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Missing Authorization header",
			})
		}

		// 解析 Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid Authorization format. Expected: Bearer <token>",
			})
		}

		// 验证 token
		if parts[1] != config.Cfg.Server.APIToken {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "Invalid token",
			})
		}

		return c.Next()
	}
}
