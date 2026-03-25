#!/bin/bash
#
# mail2dingtalk systemd 服务安装脚本
# 用法：./install-service.sh install|uninstall
#

set -e

SERVICE_NAME="mail2dingtalk"
SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}.service"
CONFIG_DIR="/etc/mail2dingtalk"
CONFIG_FILE="${CONFIG_DIR}/config.yaml"
BIN_FILE="/usr/local/bin/${SERVICE_NAME}"
DATA_DIR="/var/lib/mail2dingtalk"
LOG_DIR="/var/log/mail2dingtalk"

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_root() {
    if [[ $EUID -ne 0 ]]; then
        log_error "此脚本必须以 root 权限运行"
        exit 1
    fi
}

create_service_file() {
    cat > "${SERVICE_FILE}" << EOF
[Unit]
Description=Mail to DingTalk Service
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=${DATA_DIR}
ExecStart=${BIN_FILE}
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF
}

install() {
    log_info "开始安装 ${SERVICE_NAME}..."

    # 检查二进制文件
    if [[ ! -f "${SCRIPT_DIR}/mail2dingtalk" ]]; then
        log_error "未找到 mail2dingtalk 二进制文件，请先编译：go build -o mail2dingtalk"
        exit 1
    fi

    # 创建目录
    log_info "创建系统目录..."
    mkdir -p "${CONFIG_DIR}"
    mkdir -p "${DATA_DIR}/data/emails"
    mkdir -p "${DATA_DIR}/tmp/attachments"
    mkdir -p "${LOG_DIR}"

    # 复制二进制文件
    log_info "复制二进制文件到 ${BIN_FILE}..."
    cp "${SCRIPT_DIR}/mail2dingtalk" "${BIN_FILE}"
    chmod +x "${BIN_FILE}"

    # 处理配置文件
    if [[ -f "${CONFIG_FILE}" ]]; then
        BACKUP_FILE="${CONFIG_FILE}.backup.$(date +%Y%m%d%H%M%S)"
        log_warn "配置文件已存在，备份到 ${BACKUP_FILE}"
        cp "${CONFIG_FILE}" "${BACKUP_FILE}"
    fi

    if [[ ! -f "${CONFIG_FILE}" ]]; then
        log_info "复制配置文件到 ${CONFIG_FILE}..."
        cp "${SCRIPT_DIR}/config.yaml" "${CONFIG_FILE}"
        
        # 更新配置文件中的路径为系统路径
        sed -i "s|email_dir:.*|email_dir: \"${DATA_DIR}/data/emails\"|" "${CONFIG_FILE}"
        sed -i "s|attachment_dir:.*|attachment_dir: \"${DATA_DIR}/tmp/attachments\"|" "${CONFIG_FILE}"
        sed -i "s|file:.*|file: \"${LOG_DIR}/app.log\"|" "${CONFIG_FILE}"
    else
        log_warn "保留现有配置文件，未覆盖"
    fi

    # 创建 systemd 服务文件
    log_info "创建 systemd 服务文件..."
    create_service_file

    # 重载 systemd
    log_info "重载 systemd 配置..."
    systemctl daemon-reload

    # 启用服务
    log_info "启用服务..."
    systemctl enable "${SERVICE_NAME}"

    log_info "安装完成！"
    echo ""
    log_info "使用以下命令管理服务："
    echo "  systemctl start ${SERVICE_NAME}      # 启动服务"
    echo "  systemctl stop ${SERVICE_NAME}       # 停止服务"
    echo "  systemctl restart ${SERVICE_NAME}    # 重启服务"
    echo "  systemctl status ${SERVICE_NAME}     # 查看状态"
    echo "  journalctl -u ${SERVICE_NAME} -f     # 查看日志"
    echo ""
    log_warn "请记得编辑 ${CONFIG_FILE} 配置钉钉机器人 webhook 地址！"
}

uninstall() {
    log_info "开始卸载 ${SERVICE_NAME}..."

    # 停止并禁用服务
    if systemctl is-active --quiet "${SERVICE_NAME}"; then
        log_info "停止服务..."
        systemctl stop "${SERVICE_NAME}"
    fi

    if systemctl is-enabled --quiet "${SERVICE_NAME}"; then
        log_info "禁用服务..."
        systemctl disable "${SERVICE_NAME}"
    fi

    # 删除服务文件
    if [[ -f "${SERVICE_FILE}" ]]; then
        log_info "删除 systemd 服务文件..."
        rm -f "${SERVICE_FILE}"
    fi

    # 删除二进制文件
    if [[ -f "${BIN_FILE}" ]]; then
        log_info "删除二进制文件..."
        rm -f "${BIN_FILE}"
    fi

    # 询问是否删除配置和数据
    echo ""
    read -p "是否删除配置文件和数据？(y/N): " -n 1 -r
    echo ""
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        if [[ -d "${CONFIG_DIR}" ]]; then
            log_info "删除配置目录..."
            rm -rf "${CONFIG_DIR}"
        fi
        if [[ -d "${DATA_DIR}" ]]; then
            log_info "删除数据目录..."
            rm -rf "${DATA_DIR}"
        fi
        if [[ -d "${LOG_DIR}" ]]; then
            log_info "删除日志目录..."
            rm -rf "${LOG_DIR}"
        fi
    else
        log_warn "保留配置文件和数据"
        log_info "配置目录：${CONFIG_DIR}"
        log_info "数据目录：${DATA_DIR}"
        log_info "日志目录：${LOG_DIR}"
    fi

    # 重载 systemd
    systemctl daemon-reload

    log_info "卸载完成！"
}

show_usage() {
    echo "用法：$0 {install|uninstall}"
    echo ""
    echo "命令:"
    echo "  install    安装 mail2dingtalk 服务"
    echo "  uninstall  卸载 mail2dingtalk 服务"
    echo ""
    exit 1
}

# 主程序
check_root

case "${1:-}" in
    install)
        install
        ;;
    uninstall)
        uninstall
        ;;
    *)
        show_usage
        ;;
esac
