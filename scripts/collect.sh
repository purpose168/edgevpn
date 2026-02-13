#!/bin/bash
# 收集调试数据
# 要与 edgevpn 一起运行，请使用 --debug 和 --api 标志启动
# 注意：需要 https://github.com/whyrusleeping/stackparse 来解析 goroutine 调试堆栈

# 创建 collect 目录用于存储收集的数据
mkdir collect

# 初始化计数器，用于为收集的文件命名
((count=1))

# 无限循环，定期收集性能分析数据
while true; do
	# 计数器递增
	(( count = count + 1))
	
	# 收集堆内存分析数据
	# pprof 是 Go 语言的性能分析工具，heap 端点提供堆内存使用情况
	curl http://localhost:8080/debug/pprof/heap > collect/heap$count
	
	# 收集 goroutine 信息（二进制格式）
	# goroutine 是 Go 语言的轻量级线程，此端点提供当前所有 goroutine 的堆栈信息
	curl 'http://localhost:8080/debug/pprof/goroutine' > collect/goroutine$count
	
	# 收集 goroutine 调试信息（详细文本格式）
	# debug=2 参数表示以人类可读的文本格式输出详细的 goroutine 信息
	curl 'http://localhost:8080/debug/pprof/goroutine?debug=2' > collect/goroutine_debug_$count
	
	# 使用 stackparse 工具解析 goroutine 调试堆栈并生成摘要报告
	# --summary 参数生成简洁的摘要信息，便于快速分析
	stackparse --summary collect/goroutine_debug_$count > collect/goroutine_debug_${count}_summary
	
	# 等待 60 秒后继续下一次收集
	sleep 60
done
	
