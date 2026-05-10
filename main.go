package main

import (
	"log"

	"github.com/LightBulbClub/sugarcane/internal/analysis"
	"github.com/LightBulbClub/sugarcane/internal/config"
	"github.com/LightBulbClub/sugarcane/internal/service"
)

func main() {
	// 1. 加载配置
	if err := config.Load("config.toml"); err != nil {
		panic("加载配置失败: " + err.Error())
	}
	log.Printf("配置加载完成: InfluxDB=%s, Database=%s", config.Cfg.Influx.URL, config.Cfg.Influx.Database)

	// 2. 初始化 InfluxDB 连接
	influxService, err := service.NewInfluxDBService()
	if err != nil {
		log.Fatalf("Failed to initialize InfluxDB: %v", err)
	}
	defer influxService.Close()

	// 3. 启动实时分析和告警引擎 (协程)
	alertEngine := analysis.NewAlertEngine()
	alertEngine.Start()

	// 4. 启动 HTTP 服务器 (阻塞主协程)
	httpServer := service.NewHTTPServer(influxService)
	httpServer.Start()
}
