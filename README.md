# SugarCane 智能司机监测系统服务器

SugarCane 是一个基于 Go 的司机状态监测服务器，配合 glucose 手环设备使用，实时监测司机的心率和加速度数据。

## 项目结构

```
sugarcane/
├── main.go                    # 程序入口
├── config.toml                # 配置文件
├── internal/
│   ├── analysis/
│   │   └── engine.go          # 实时分析和告警引擎
│   ├── config/
│   │   └── config.go          # 配置管理
│   ├── handler/
│   │   └── data.go            # HTTP 请求处理器
│   ├── model/
│   │   └── band.go            # 数据模型定义
│   └── service/
│       ├── influxdb.go        # InfluxDB 服务
│       └── server.go          # HTTP 服务器
```

## 功能特性

- **多格式数据接收**：兼容 glucose MicroPython 手环数据格式和标准格式
- **InfluxDB3 存储**：高效存储时间序列数据
- **实时分析引擎**：心率异常检测、加速度异常检测
- **告警系统**：支持告警静默期，避免重复告警
- **RESTful API**：标准的 HTTP 接口

## API 端点

### POST /data/upload
接收 glucose 手环数据上传。

**请求数据格式：**
```json
{
  "acceleration": [0.1, 0.2, 9.8],
  "heart": [72, 98.5],
  "driverId": "001521"
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| acceleration | [3]float64 | 三轴加速度 [X, Y, Z]（单位：g） |
| heart | [2]float64 | [心率(bpm), 血氧饱和度(%)] |
| driverId | string | 司机唯一标识 |

**响应：**
```json
{
  "message": "Data received successfully",
  "driver_id": "001521",
  "heart_rate": 72
}
```

### GET /health
健康检查端点。

## 快速开始

1. **配置 InfluxDB**
   编辑 `config.toml` 文件，配置 InfluxDB 连接信息。

2. **运行服务器**
   ```bash
   go run main.go
   ```

3. **配置 glucose 手环**
   修改 glucose 项目的 `src/main.py` 中的服务器地址。

## 配置说明

| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| influx.url | InfluxDB 地址 | http://localhost:8181 |
| influx.database | 数据库名称 | sugarcane |
| server.listen_port | 监听端口 | :3000 |
| analysis.heart_rate_min | 心率下限 | 50 |
| analysis.heart_rate_max | 心率上限 | 100 |
| analysis.accel_threshold | 加速度阈值 | 2.0 |
| analysis.alert_cooldown | 告警冷却时间 | 30s |

## 依赖

- Go 1.21+
- InfluxDB 3.x
- Fiber v2
