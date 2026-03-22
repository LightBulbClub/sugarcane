package main

import (
	"github.com/LightBulbClub/sugarcane/analysis"
	"github.com/LightBulbClub/sugarcane/config"
	"github.com/LightBulbClub/sugarcane/service"
)

func main() {
	// 0. 加载配置
	if err := config.Load("config.toml"); err != nil {
		panic("加载配置失败: " + err.Error())
	}
	// 1. 初始化数据库连接
	service.InitInfluxDB()
	defer service.CloseInfluxDB()

	// 2. 启动实时分析和告警引擎 (协程)
	analysis.StartAlertEngine()

	// 3. 启动 Fiber Web 服务器 (阻塞主协程)
	service.StartServer()
}
