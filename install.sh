#!/bin/sh
set -e
set -o noglob

github_version() {
    set +e
    # 获取最新版本号，如果失败则返回默认版本 v30.2
    curl -s https://api.github.com/repos/mudler/edgevpn/releases/latest | \
    grep tag_name | \
    awk '{ print $2 }' | \
    sed -e 's/"//g' -e 's/,//g' || echo "v30.2"
    set -e
}

# 下载工具，默认使用 curl
DOWNLOADER=${DOWNLOADER:-curl}
# 版本号，默认从 GitHub API 获取
VERSION=${VERSION:-$(github_version)}

info()
{
    echo '[INFO] ' "$@"
}

warn()
{
    echo '[WARN] ' "$@" >&2
}

fatal()
{
    echo '[ERROR] ' "$@" >&2
    exit 1
}


# 检测系统架构
detect_arch() {
    if [ -z "$ARCH" ]; then
        ARCH=$(uname -m)
    fi
    case $ARCH in
        i386)
            ARCH=i386
            ;;
        amd64|x86_64)
            ARCH=x86_64
            ;;
        arm64|aarch64)
            ARCH=arm64
            ;;
        arm*)
            ARCH=armv6
            ;;
        *)
            fatal "不支持的架构 $ARCH"
    esac
}

# 检测操作系统平台
detect_platform() {
    if [ -z "$OS" ]; then
        OS=$(uname -o)
    fi
    case $OS in
        *Linux)
            OS=Linux
            ;;
        *)
            fatal "不支持的平台 $OS"
    fi
}

# 验证环境并设置变量
verify_env() {

    detect_arch
    detect_platform

    # 检测是否有 openrc 服务管理器
    if [ -x /sbin/openrc-run ]; then
        HAS_OPENRC=true
    fi
    # 检测是否有 systemd 服务管理器
    if [ -x /bin/systemctl ] || type systemctl > /dev/null 2>&1; then
        HAS_SYSTEMD=true
    fi

    # 设置 sudo 命令，如果已经是 root 则不需要
    SUDO=sudo
    if [ $(id -u) -eq 0 ]; then
        SUDO=
    fi

    # 设置二进制文件安装目录
    if [ -n "${INSTALL_BIN_DIR}" ]; then
        BIN_DIR=${INSTALL_BIN_DIR}
    else
        BIN_DIR=/usr/local/bin
        # 测试目录是否可写
        if ! $SUDO sh -c "touch ${BIN_DIR}/ro-test && rm -rf ${BIN_DIR}/ro-test"; then
            if [ -d /opt/bin ]; then
                BIN_DIR=/opt/bin
            fi
        fi
    fi

}

# 设置系统服务
setup_service() {
    # 设置 systemd 服务目录
    if [ -n "${INSTALL_SYSTEMD_DIR}" ]; then
        SYSTEMD_DIR="${INSTALL_SYSTEMD_DIR}"
    else
        SYSTEMD_DIR=/etc/systemd/system
    fi

    # 如果有 systemd
    if [ "${HAS_SYSTEMD}" = true ]; then
        FILE_SERVICE=${SYSTEMD_DIR}/edgevpn@.service
        $SUDO tee $FILE_SERVICE >/dev/null << EOF
[Unit]
Description=EdgeVPN 守护进程
After=network.target

[Service]
EnvironmentFile=/etc/systemd/system.conf.d/edgevpn-%i.env
LimitNOFILE=49152
ExecStartPre=-/bin/sh -c "sysctl -w net.core.rmem_max=2500000"
ExecStart=$BIN_DIR/edgevpn
Restart=always

[Install]
WantedBy=multi-user.target
EOF
    # 如果有 openrc
    elif [ "${HAS_OPENRC}" = true ]; then
        $SUDO tee /etc/init.d/edgevpn >/dev/null << EOF
#!/sbin/openrc-run
depend() {
    after network-online
}

supervisor=supervise-daemon
name=edgevpn
command="${BIN_DIR}/edgevpn"
command_args="$(escape_dq "edgevpn")
    >>${LOG_FILE} 2>&1"
output_log=${LOG_FILE}
error_log=${LOG_FILE}
pidfile="/var/run/edgevpn.pid"
respawn_delay=5
respawn_max=0
set -o allexport
if [ -f /etc/environment ]; then source /etc/environment; fi
if [ -f /etc/edgevpn.env ]; then source /etc/edgevpn.env; fi
set +o allexport
EOF
    fi

}

# 下载文件
download() {
    [ $# -eq 2 ] || fatal 'download 需要恰好 2 个参数'

    case $DOWNLOADER in
        curl)
            curl -o $1 -sfL $2
            ;;
        wget)
            wget -qO $1 $2
            ;;
        *)
            fatal "不正确的可执行文件 '$DOWNLOADER'"
            ;;
    esac

    # 如果下载失败则退出
    [ $? -eq 0 ] || fatal '下载失败'
}

# 安装主函数
install() {
    info "架构: $ARCH. 操作系统: $OS 版本: $VERSION (\\$VERSION)"

    # 创建临时目录
    TMP_DIR=$(mktemp -d -t edgevpn-install.XXXXXXXXXX)

    # 下载预编译二进制文件
    download $TMP_DIR/out.tar.gz https://github.com/mudler/edgevpn/releases/download/$VERSION/edgevpn-$VERSION-$OS-$ARCH.tar.gz

    # 解压文件
    tar xvf $TMP_DIR/out.tar.gz -C $TMP_DIR

    # 复制二进制文件到安装目录
    $SUDO cp -rf $TMP_DIR/edgevpn $BIN_DIR/

    # 清理临时目录
    rm -rf $TMP_DIR

    # TODO: 设置网络连接的环境文件
}

# 执行验证、设置服务和安装
verify_env
setup_service
install