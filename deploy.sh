#!/usr/bin/env bash
# SugarCane 一键部署脚本
# 用于安装 InfluxDB 3 Core 和启动 sugarcane 服务

set -euo pipefail

# ==================== 配置 ====================
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INSTALL_DIR="${SCRIPT_DIR}/.local"
INFLUXDB_DIR="${INSTALL_DIR}/influxdb3"
SUGARCANE_BIN="${INSTALL_DIR}/sugarcane"
PID_DIR="${INSTALL_DIR}/pids"
LOG_DIR="${INSTALL_DIR}/logs"
DATA_DIR="${INSTALL_DIR}/data"

# InfluxDB 3 Core 版本
INFLUXDB3_VERSION="3.9.2"
INFLUXDB3_PORT=8181

# sugarcane 配置
SUGARCANE_PORT=3000

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# ==================== 工具函数 ====================

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

check_command() {
    command -v "$1" &> /dev/null
}

# ==================== 环境检查 ====================

check_environment() {
    log_step "检查系统环境..."

    # 检查操作系统
    if [[ "$(uname)" != "Linux" ]]; then
        log_error "此脚本仅支持 Linux 系统"
        exit 1
    fi

    # 检查架构
    ARCH="$(uname -m)"
    case "${ARCH}" in
        x86_64|amd64)
            INFLUXDB3_ARCH="amd64"
            ;;
        aarch64|arm64)
            INFLUXDB3_ARCH="arm64"
            ;;
        *)
            log_error "不支持的架构: ${ARCH}"
            exit 1
            ;;
    esac

    # 检查必要工具
    for cmd in curl tar; do
        if ! check_command "$cmd"; then
            log_error "缺少必要工具: $cmd"
            exit 1
        fi
    done

    # 检查 Go
    if ! check_command go; then
        log_error "未安装 Go，请先安装 Go 1.21+"
        exit 1
    fi

    GO_VERSION=$(go version | grep -oP 'go\K[0-9]+\.[0-9]+')
    log_info "Go 版本: ${GO_VERSION}"

    log_info "系统架构: ${ARCH}"
    log_info "环境检查通过"
}

# ==================== InfluxDB 3 安装 ====================

install_influxdb3() {
    log_step "安装 InfluxDB 3 Core..."

    mkdir -p "${INFLUXDB_DIR}"

    local binary="${INFLUXDB_DIR}/influxdb3"
    if [[ -f "${binary}" ]]; then
        log_info "InfluxDB 3 已存在，跳过下载"
        return 0
    fi

    local url="https://dl.influxdata.com/influxdb/releases/influxdb3-core-${INFLUXDB3_VERSION}_linux_${INFLUXDB3_ARCH}.tar.gz"
    local tmp_file="/tmp/influxdb3-${INFLUXDB3_VERSION}.tar.gz"

    log_info "下载 InfluxDB 3 Core v${INFLUXDB3_VERSION}..."
    log_info "URL: ${url}"

    if ! curl -L -o "${tmp_file}" "${url}" --progress-bar; then
        log_error "下载失败"
        exit 1
    fi

    log_info "解压中..."
    tar -xzf "${tmp_file}" -C "${INFLUXDB_DIR}" --strip-components=1
    rm -f "${tmp_file}"

    chmod +x "${binary}"

    if [[ ! -f "${binary}" ]]; then
        log_error "InfluxDB 3 二进制文件不存在: ${binary}"
        exit 1
    fi

    log_info "InfluxDB 3 安装完成: ${binary}"
}

# ==================== InfluxDB 3 配置和启动 ====================

setup_influxdb3() {
    log_step "配置 InfluxDB 3..."

    mkdir -p "${DATA_DIR}/influxdb3" "${LOG_DIR}" "${PID_DIR}"

    # 创建启动脚本（本地开发环境禁用认证）
    cat > "${INFLUXDB_DIR}/start.sh" << 'INFLUXDB_START_EOF'
#!/usr/bin/env bash
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
exec "${SCRIPT_DIR}/influxdb3" serve \
    --node-id=sugarcane-local \
    --object-store=file \
    --data-dir="${SCRIPT_DIR}/../data/influxdb3" \
    --http-bind=0.0.0.0:8181 \
    --without-auth
INFLUXDB_START_EOF

    chmod +x "${INFLUXDB_DIR}/start.sh"
    log_info "InfluxDB 3 配置完成"
}

start_influxdb3() {
    log_step "启动 InfluxDB 3..."

    local pid_file="${PID_DIR}/influxdb3.pid"

    # 检查是否已运行
    if [[ -f "${pid_file}" ]]; then
        local pid
        pid=$(cat "${pid_file}")
        if kill -0 "${pid}" 2>/dev/null; then
            log_info "InfluxDB 3 已在运行 (PID: ${pid})"
            return 0
        fi
        rm -f "${pid_file}"
    fi

    # 启动 InfluxDB 3
    nohup "${INFLUXDB_DIR}/start.sh" > "${LOG_DIR}/influxdb3.log" 2>&1 &
    local pid=$!
    echo "${pid}" > "${pid_file}"

    # 等待启动
    log_info "等待 InfluxDB 3 启动..."
    local max_wait=30
    local count=0
    while ! curl -s "http://localhost:${INFLUXDB3_PORT}/health" > /dev/null 2>&1; do
        if [[ ${count} -ge ${max_wait} ]]; then
            log_error "InfluxDB 3 启动超时"
            cat "${LOG_DIR}/influxdb3.log"
            exit 1
        fi
        sleep 1
        count=$((count + 1))
    done

    log_info "InfluxDB 3 启动成功 (PID: ${pid})"
    log_info "地址: http://localhost:${INFLUXDB3_PORT}"
}

# ==================== sugarcane 编译和启动 ====================

build_sugarcane() {
    log_step "编译 sugarcane..."

    cd "${SCRIPT_DIR}"

    # 下载依赖
    log_info "下载 Go 依赖..."
    go mod download

    # 编译
    log_info "编译中..."
    go build -o "${SUGARCANE_BIN}" .

    if [[ ! -f "${SUGARCANE_BIN}" ]]; then
        log_error "编译失败"
        exit 1
    fi

    log_info "编译完成: ${SUGARCANE_BIN}"
}

update_sugarcane_config() {
    log_step "更新 sugarcane 配置..."

    local config_file="${SCRIPT_DIR}/config.toml"

    # 如果配置文件不存在，创建默认配置
    if [[ ! -f "${config_file}" ]]; then
        local api_token
        api_token=$(openssl rand -base64 32 2>/dev/null || python3 -c 'import secrets; print(secrets.token_urlsafe(32))')

        cat > "${config_file}" << CONFIG_EOF
# SugarCane 配置文件

[influx]
url = "http://localhost:${INFLUXDB3_PORT}"
token = ""
org = ""
bucket = "driver_monitoring"
measurement = "band_data"
database = "sugarcane"

[server]
listen_port = ":${SUGARCANE_PORT}"
api_token = "${api_token}"

[analysis]
heart_rate_min = 50
heart_rate_max = 100
accel_threshold = 2.0
alert_cooldown = "30s"
CONFIG_EOF
        log_info "已创建默认配置文件: ${config_file}"
    else
        log_info "配置文件已存在: ${config_file}"
    fi
}

start_sugarcane() {
    log_step "启动 sugarcane..."

    local pid_file="${PID_DIR}/sugarcane.pid"

    # 检查是否已运行
    if [[ -f "${pid_file}" ]]; then
        local pid
        pid=$(cat "${pid_file}")
        if kill -0 "${pid}" 2>/dev/null; then
            log_info "sugarcane 已在运行 (PID: ${pid})"
            return 0
        fi
        rm -f "${pid_file}"
    fi

    # 启动 sugarcane
    cd "${SCRIPT_DIR}"
    nohup "${SUGARCANE_BIN}" > "${LOG_DIR}/sugarcane.log" 2>&1 &
    local pid=$!
    echo "${pid}" > "${pid_file}"

    # 等待启动
    sleep 2
    if kill -0 "${pid}" 2>/dev/null; then
        log_info "sugarcane 启动成功 (PID: ${pid})"
        log_info "地址: http://localhost:${SUGARCANE_PORT}"
    else
        log_error "sugarcane 启动失败"
        cat "${LOG_DIR}/sugarcane.log"
        exit 1
    fi
}

# ==================== 服务管理 ====================

stop_service() {
    local name=$1
    local pid_file="${PID_DIR}/${name}.pid"

    if [[ ! -f "${pid_file}" ]]; then
        log_warn "${name} 未运行"
        return 0
    fi

    local pid
    pid=$(cat "${pid_file}")
    if kill -0 "${pid}" 2>/dev/null; then
        log_info "停止 ${name} (PID: ${pid})..."
        kill "${pid}"
        sleep 2
        if kill -0 "${pid}" 2>/dev/null; then
            kill -9 "${pid}" 2>/dev/null || true
        fi
    fi
    rm -f "${pid_file}"
    log_info "${name} 已停止"
}

show_status() {
    echo ""
    echo "=== SugarCane 服务状态 ==="
    echo ""

    # InfluxDB 3 状态
    local influx_pid_file="${PID_DIR}/influxdb3.pid"
    if [[ -f "${influx_pid_file}" ]] && kill -0 "$(cat "${influx_pid_file}")" 2>/dev/null; then
        echo -e "InfluxDB 3:  ${GREEN}运行中${NC} (PID: $(cat "${influx_pid_file}"))"
        echo "             http://localhost:${INFLUXDB3_PORT}"
    else
        echo -e "InfluxDB 3:  ${RED}未运行${NC}"
    fi

    # sugarcane 状态
    local sugarcane_pid_file="${PID_DIR}/sugarcane.pid"
    if [[ -f "${sugarcane_pid_file}" ]] && kill -0 "$(cat "${sugarcane_pid_file}")" 2>/dev/null; then
        echo -e "sugarcane:   ${GREEN}运行中${NC} (PID: $(cat "${sugarcane_pid_file}"))"
        echo "             http://localhost:${SUGARCANE_PORT}"
    else
        echo -e "sugarcane:   ${RED}未运行${NC}"
    fi

    echo ""
    echo "日志目录: ${LOG_DIR}"
    echo ""
}

show_logs() {
    local service="${1:-all}"

    case "${service}" in
        influxdb|influxdb3)
            if [[ -f "${LOG_DIR}/influxdb3.log" ]]; then
                tail -f "${LOG_DIR}/influxdb3.log"
            else
                log_error "InfluxDB 3 日志不存在"
            fi
            ;;
        sugarcane)
            if [[ -f "${LOG_DIR}/sugarcane.log" ]]; then
                tail -f "${LOG_DIR}/sugarcane.log"
            else
                log_error "sugarcane 日志不存在"
            fi
            ;;
        all)
            if [[ -f "${LOG_DIR}/influxdb3.log" ]] || [[ -f "${LOG_DIR}/sugarcane.log" ]]; then
                tail -f "${LOG_DIR}"/*.log
            else
                log_error "没有日志文件"
            fi
            ;;
        *)
            log_error "未知服务: ${service}"
            echo "用法: $0 logs [influxdb3|sugarcane|all]"
            ;;
    esac
}

# ==================== 主命令 ====================

cmd_install() {
    echo ""
    echo "=== SugarCane 一键部署 ==="
    echo ""

    check_environment
    install_influxdb3
    setup_influxdb3
    build_sugarcane
    update_sugarcane_config

    echo ""
    log_info "安装完成！"
    echo ""
    echo "运行以下命令启动服务:"
    echo "  ./deploy.sh start"
    echo ""
}

cmd_start() {
    echo ""
    echo "=== 启动 SugarCane 服务 ==="
    echo ""

    # 检查是否已安装
    if [[ ! -f "${INFLUXDB_DIR}/influxdb3" ]]; then
        log_error "InfluxDB 3 未安装，请先运行: ./deploy.sh install"
        exit 1
    fi

    if [[ ! -f "${SUGARCANE_BIN}" ]]; then
        log_error "sugarcane 未编译，请先运行: ./deploy.sh install"
        exit 1
    fi

    start_influxdb3
    start_sugarcane

    echo ""
    show_status
}

cmd_stop() {
    echo ""
    echo "=== 停止 SugarCane 服务 ==="
    echo ""

    stop_service "sugarcane"
    stop_service "influxdb3"

    echo ""
    log_info "所有服务已停止"
}

cmd_restart() {
    cmd_stop
    cmd_start
}

cmd_status() {
    show_status
}

cmd_logs() {
    show_logs "${1:-all}"
}

cmd_uninstall() {
    echo ""
    echo "=== 卸载 SugarCane ==="
    echo ""

    # 停止服务
    cmd_stop

    # 删除安装目录
    if [[ -d "${INSTALL_DIR}" ]]; then
        log_info "删除安装目录: ${INSTALL_DIR}"
        rm -rf "${INSTALL_DIR}"
    fi

    echo ""
    log_info "卸载完成（配置文件 config.toml 已保留）"
}

cmd_help() {
    echo ""
    echo "SugarCane 部署脚本"
    echo ""
    echo "用法: $0 <命令>"
    echo ""
    echo "命令:"
    echo "  install     安装 InfluxDB 3 和编译 sugarcane"
    echo "  start       启动所有服务"
    echo "  stop        停止所有服务"
    echo "  restart     重启所有服务"
    echo "  status      查看服务状态"
    echo "  logs        查看日志 (可指定: influxdb3, sugarcane, all)"
    echo "  uninstall   停止服务并删除安装文件"
    echo "  help        显示此帮助信息"
    echo ""
    echo "示例:"
    echo "  $0 install          # 首次安装"
    echo "  $0 start            # 启动服务"
    echo "  $0 logs sugarcane   # 查看 sugarcane 日志"
    echo "  $0 status           # 查看状态"
    echo ""
}

# ==================== 入口 ====================

main() {
    local command="${1:-help}"

    case "${command}" in
        install)
            cmd_install
            ;;
        start)
            cmd_start
            ;;
        stop)
            cmd_stop
            ;;
        restart)
            cmd_restart
            ;;
        status)
            cmd_status
            ;;
        logs)
            cmd_logs "${2:-all}"
            ;;
        uninstall)
            cmd_uninstall
            ;;
        help|--help|-h)
            cmd_help
            ;;
        *)
            log_error "未知命令: ${command}"
            cmd_help
            exit 1
            ;;
    esac
}

main "$@"
