#!/bin/bash
# =============================================================================
# EdgeVPN 测试脚本
# =============================================================================
# 用途：自动化运行 EdgeVPN 项目的测试套件
# 作者：EdgeVPN 开发团队
# 创建日期：见版本控制历史
# 联系方式：purpose168@outlook.com
# =============================================================================
# 使用方法：
#   ./tests.sh
# 
# 前提条件：
#   - 已安装 Go 语言环境
#   - 已编译 edgevpn 可执行文件
#   - 项目依赖已正确配置
# =============================================================================

# 设置 Shell 执行选项
# -e：当命令返回非零状态码时立即退出脚本（遇到错误即停止）
# -x：在执行命令前打印该命令（便于调试和追踪执行流程）
set -ex

# 安装 Ginkgo 测试框架
# go install：编译并安装 Go 包或可执行文件
# -mod=mod：使用 Go modules 模式管理依赖，自动更新 go.mod 文件
# github.com/onsi/ginkgo/v2/ginkgo：Ginkgo 是一个流行的 Go BDD 测试框架
# 安装后，ginkgo 命令将可用作测试运行器
go install -mod=mod github.com/onsi/ginkgo/v2/ginkgo

# 启动 EdgeVPN API 服务（后台运行）
# ./edgevpn api：运行 edgevpn 程序的 api 子命令
# &：将进程放入后台执行，允许脚本继续执行后续命令
# 注意：此服务将在 localhost:8080 上监听请求
./edgevpn api &

# 设置测试实例环境变量
# export：导出环境变量，使其在当前 Shell 及其子进程中可用
# TEST_INSTANCE：指定测试目标 API 服务的地址
# http://localhost:8080：本地测试服务器的默认端口
export TEST_INSTANCE="http://localhost:8080"

# 运行 Ginkgo 测试套件
# ginkgo：Ginkgo 测试运行器命令
# -v：详细输出模式，显示每个测试用例的执行详情
# -r：递归模式，测试指定目录及其所有子目录中的测试套件
# --flake-attempts 5：失败重试次数，对于不稳定的测试最多重试 5 次
#   （某些测试可能因网络、时序等因素偶发性失败，重试可提高测试稳定性）
# --coverprofile=coverage.txt：生成代码覆盖率报告文件
# --covermode=atomic：覆盖率统计模式，使用原子操作确保并发安全
# --race：启用竞态检测器，检测并发程序中的数据竞争问题
# ./pkg/... ./api/...：指定测试路径
#   - ./pkg/...：测试 pkg 目录下的所有包
#   - ./api/...：测试 api 目录下的所有包
ginkgo -v -r --flake-attempts 5 --coverprofile=coverage.txt --covermode=atomic --race ./pkg/... ./api/...