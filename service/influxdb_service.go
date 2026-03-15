package service

import (
	"context"
	"log"

	"github.com/LightBulbClub/sugarcane/config"
	"github.com/LightBulbClub/sugarcane/data"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

// InfluxClient 暴露的客户端实例
var InfluxClient influxdb2.Client

// InitInfluxDB 初始化 InfluxDB 连接
func InitInfluxDB() {
	log.Printf("Connecting to InfluxDB at %s...", config.InfluxURL)

	// 将客户端赋值给全局 App 结构体
	data.GlobalApp.InfluxClient = influxdb2.NewClient(config.InfluxURL, config.InfluxToken)

	// 检查连接状态
	_, err := data.GlobalApp.InfluxClient.Health(context.Background())
	if err != nil {
		log.Fatalf("Failed to connect to InfluxDB: %v", err)
	}
	log.Println("Successfully connected to InfluxDB!")
}

// CloseInfluxDB 关闭 InfluxDB 连接
func CloseInfluxDB() {
	if data.GlobalApp.InfluxClient != nil {
		data.GlobalApp.InfluxClient.Close()
		log.Println("InfluxDB connection closed.")
	}
}
