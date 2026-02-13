#!/bin/bash
# =============================================================================
# 脚本名称：serve.sh
# 用途：启动 Hugo 本地开发服务器
# 作者：purpose168@outlook.com
# 创建日期：2026-02-13
# =============================================================================

# 设置错误时立即退出
# -e 选项：当任何命令返回非零状态码时，脚本立即退出
set -e

# 定义 Hugo 二进制文件的存放路径
# ROOT_DIR 是环境变量，表示项目根目录
binpath="${ROOT_DIR}/bin"

# 检查 Hugo 可执行文件是否存在
# 如果不存在，则下载并安装
if [ ! -e "${binpath}/hugo" ];
then
    # 如果 bin 目录不存在，则创建该目录
    # -p 选项：递归创建目录，如果父目录不存在也会创建
    [[ ! -d "${binpath}" ]] && mkdir -p "${binpath}"

    # 从 GitHub 下载指定版本的 Hugo 扩展版
    # HUGO_VERSION 和 HUGO_PLATFORM 是环境变量，分别指定版本号和平台类型
    # wget：下载工具，-O 指定输出文件名
    wget https://github.com/gohugoio/hugo/releases/download/v"${HUGO_VERSION}"/hugo_extended_"${HUGO_VERSION}"_"${HUGO_PLATFORM}".tar.gz -O "$binpath"/hugo.tar.gz

    # 解压 Hugo 压缩包到 bin 目录
    # tar：归档工具
    # -x：解压
    # -v：显示详细过程
    # -f：指定文件
    # -C：指定解压目标目录
    tar -xvf "$binpath"/hugo.tar.gz -C "${binpath}"

    # 删除压缩包文件，节省磁盘空间
    rm -rf "$binpath"/hugo.tar.gz

    # 为 Hugo 可执行文件添加执行权限
    # chmod：修改文件权限
    # +x：添加执行权限
    chmod +x "$binpath"/hugo
fi

# 启动 Hugo 本地开发服务器
# --baseURL：设置网站的基础 URL
# -s：指定站点根目录
# serve：启动本地开发服务器，默认监听 http://localhost:1313
"${binpath}/hugo" --baseURL="$BASE_URL" -s "$ROOT_DIR" serve
