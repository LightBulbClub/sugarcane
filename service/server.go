package service

import (
	"log"

	"github.com/LightBulbClub/sugarcane/config"
	"github.com/LightBulbClub/sugarcane/handler"

	"github.com/gofiber/fiber/v2"
)

// StartServer 初始化并启动 Fiber HTTP 服务
func StartServer() {
	app := fiber.New(fiber.Config{
		Prefork: false, // 简化示例，生产环境可开启
	})

	// 注册路由
	app.Post("/data/upload", handler.DataUploadHandler)

	// 启动服务
	log.Printf("Fiber server starting on %s...", config.ListenPort)
	log.Fatal(app.Listen(config.ListenPort))
}
