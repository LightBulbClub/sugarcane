package service

import (
	"log"

	"github.com/LightBulbClub/sugarcane/config"
	"github.com/LightBulbClub/sugarcane/data"

	"github.com/InfluxCommunity/influxdb3-go/v2/influxdb3"
)

// InitInfluxDB 初始化 InfluxDB 连接
func InitInfluxDB() {
	log.Printf("Connecting to InfluxDB at %s...", config.Cfg.Influx.URL)

	// 将客户端赋值给全局 App 结构体
	client, err := influxdb3.New(
		influxdb3.ClientConfig{
			Token:    config.Cfg.Influx.Token,
			Host:     config.Cfg.Influx.URL,
			Database: config.Cfg.Influx.Database,
		})
	if err != nil {
		log.Fatalf("Failed to create&connect to InfluxDB client: %v", err)
	}
	data.GlobalApp.InfluxClient = client
	log.Println("Successfully connected to InfluxDB!")
}

// CloseInfluxDB 关闭 InfluxDB 连接
func CloseInfluxDB() {
	if data.GlobalApp.InfluxClient != nil {
		err := data.GlobalApp.InfluxClient.Close()
		if err != nil {
			log.Fatal("Error closing InfluxDB connection: ", err)
		}
		log.Println("InfluxDB connection closed.")
	}
}
