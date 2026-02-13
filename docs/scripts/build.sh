#!/bin/bash
# =============================================================================
# 脚本名称：build.sh
# 用途：构建 Hugo 静态网站
# 作者：purpose168@outlook.com
# 创建日期：2026-02-13
# =============================================================================

# 设置错误时立即退出
# -e 选项：当任何命令返回非零状态码时，脚本立即退出
set -e

# 设置基础 URL，如果环境变量未定义则使用默认值
# ${变量名:-默认值}：如果变量未定义或为空，则使用默认值
# 默认值为 GitHub Pages 的项目地址
BASE_URL="${BASE_URL:-https://mudler.github.io/edgevpn/}"

# 定义 Hugo 二进制文件的存放路径
# ROOT_DIR 是环境变量，表示项目根目录
binpath="${ROOT_DIR}/bin"

# 定义构建输出目录的路径
publicpath="${ROOT_DIR}/public"

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

# 清理旧的构建输出目录（如果存在）
# || true：如果目录不存在，忽略错误继续执行
rm -rf "${publicpath}" || true

# 创建新的构建输出目录
# -p 选项：递归创建目录，如果父目录不存在也会创建
[[ ! -d "${publicpath}" ]] && mkdir -p "${publicpath}"

# 注意：构建前需要安装以下依赖
# postcss-cli：PostCSS 命令行工具，用于处理 CSS
# 安装命令：sudo npm install -g postcss-cli
#

# 安装项目依赖的 npm 包（本地安装）
# npm install：安装 npm 包
# -D：作为开发依赖安装（devDependencies）
# --save：保存到 package.json 文件
# autoprefixer：自动添加 CSS 浏览器前缀
npm install -D --save autoprefixer

# 安装 PostCSS 命令行工具
# postcss-cli：PostCSS 的命令行接口
npm install -D --save postcss-cli

# 执行 Hugo 构建命令
# HUGO_ENV="production"：设置环境变量为生产环境
# --gc：启用垃圾回收，清理未使用的资源
# -b：设置基础 URL
# -s：指定站点根目录
# -d：指定输出目录
HUGO_ENV="production" "${binpath}/hugo" --gc -b "${BASE_URL}" -s "${ROOT_DIR}" -d "${publicpath}"
