package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/LightBulbClub/sugarcane/internal/config"
	"github.com/LightBulbClub/sugarcane/internal/model"

	"github.com/InfluxCommunity/influxdb3-go/v2/influxdb3"
)

// InfluxDBService 封装 InfluxDB 操作
type InfluxDBService struct {
	client *influxdb3.Client
}

// NewInfluxDBService 创建新的 InfluxDB 服务实例
func NewInfluxDBService() (*InfluxDBService, error) {
	log.Printf("Connecting to InfluxDB at %s...", config.Cfg.Influx.URL)

	client, err := influxdb3.New(influxdb3.ClientConfig{
		Token:    config.Cfg.Influx.Token,
		Host:     config.Cfg.Influx.URL,
		Database: config.Cfg.Influx.Database,
	})
	if err != nil {
		return nil, err
	}

	log.Println("Successfully connected to InfluxDB!")
	return &InfluxDBService{client: client}, nil
}

// Close 关闭 InfluxDB 连接
func (s *InfluxDBService) Close() {
	if s.client != nil {
		if err := s.client.Close(); err != nil {
			log.Printf("Error closing InfluxDB connection: %v", err)
		}
		log.Println("InfluxDB connection closed.")
	}
}

// WriteBandData 写入 BandData 到 InfluxDB
func (s *InfluxDBService) WriteBandData(ctx context.Context, d model.BandData) error {
	p := influxdb3.NewPointWithMeasurement(config.Cfg.Influx.Measurement).
		SetTag("driver_id", d.DriverID).
		SetField("heart_rate", d.HeartRate).
		SetField("spo2", d.SpO2).
		SetField("accel_x", d.AccelX).
		SetField("accel_y", d.AccelY).
		SetField("accel_z", d.AccelZ).
		SetTimestamp(d.Timestamp)

	return s.client.WritePoints(ctx, []*influxdb3.Point{p})
}

// WriteDrowsinessData 写入瞌睡检测数据到 InfluxDB
func (s *InfluxDBService) WriteDrowsinessData(ctx context.Context, d model.DrowsinessData) error {
	p := influxdb3.NewPointWithMeasurement("drowsiness").
		SetTag("driver_id", d.DriverID).
		SetField("is_drowsy", d.IsDrowsy).
		SetField("ear_value", d.EARValue).
		SetField("confidence", d.Confidence).
		SetTimestamp(d.Timestamp)

	return s.client.WritePoints(ctx, []*influxdb3.Point{p})
}

// GetDriverStatus 获取司机综合状态
func (s *InfluxDBService) GetDriverStatus(ctx context.Context, driverID string) (*model.DriverStatus, error) {
	status := &model.DriverStatus{
		DriverID: driverID,
		Alerts:   []string{},
	}

	// 查询最新的手环数据
	bandQuery := fmt.Sprintf(
		"SELECT heart_rate, spo2, time FROM %s WHERE driver_id = '%s' ORDER BY time DESC LIMIT 1",
		config.Cfg.Influx.Measurement, driverID,
	)
	bandResult, err := s.client.Query(ctx, bandQuery)
	if err != nil {
		log.Printf("Error querying band data: %v", err)
	} else {
		for bandResult.Next() {
			row := bandResult.Value()
			if hr, ok := row["heart_rate"].(int64); ok {
				status.HeartRate = int(hr)
			}
			if spo2, ok := row["spo2"].(float64); ok {
				status.SpO2 = spo2
			}
			if t, ok := row["time"].(string); ok {
				status.LastUpdate, _ = parseTime(t)
			}
		}
	}

	// 查询最新的瞌睡数据
	drowsyQuery := fmt.Sprintf(
		"SELECT is_drowsy, ear_value, confidence, time FROM drowsiness WHERE driver_id = '%s' ORDER BY time DESC LIMIT 1",
		driverID,
	)
	drowsyResult, err := s.client.Query(ctx, drowsyQuery)
	if err != nil {
		log.Printf("Error querying drowsiness data: %v", err)
	} else {
		for drowsyResult.Next() {
			row := drowsyResult.Value()
			if isDrowsy, ok := row["is_drowsy"].(bool); ok {
				status.IsDrowsy = isDrowsy
			}
			if t, ok := row["time"].(string); ok {
				parsedTime, _ := parseTime(t)
				if parsedTime.After(status.LastUpdate) {
					status.LastUpdate = parsedTime
				}
			}
		}
	}

	return status, nil
}

// parseTime 解析时间字符串
func parseTime(s string) (time.Time, error) {
	layouts := []string{
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.000Z",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse time: %s", s)
}
