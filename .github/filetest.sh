#!/bin/bash
# =============================================================================
# 文件传输测试脚本
# 用途：测试 edgevpn 的文件发送和接收功能
# 使用方法：./filetest.sh sender  (作为发送端运行)
#           ./filetest.sh         (作为接收端运行)
# 作者：purpose168@outlook.com
# 创建日期：2026-02-13
# =============================================================================

# 启动 edgevpn API 服务（后台运行）
# & 符号表示在后台运行，不阻塞当前脚本执行
./edgevpn api &

# 判断脚本参数，根据参数决定执行发送端还是接收端逻辑
if [ $1 == "sender" ]; then
    # ==================== 发送端逻辑 ====================
    
    # 创建测试文件，内容为 "test"
    # $PWD 表示当前工作目录
    # > 重定向符号，将左侧命令的输出写入右侧文件
    echo "test" > $PWD/test

    # 启动文件发送进程（后台运行）
    # --name "test"：指定文件在分布式账本中的名称标识
    # --path $PWD/test：指定要发送的本地文件路径
    ./edgevpn file-send --name "test" --path $PWD/test &

    # 初始化计数器为 240（最多等待 480 秒，即 8 分钟）
    # (( )) 是 Bash 的算术运算语法
    ((count = 240))                        
    
    # 循环等待文件传输完成
    # [[ ]] 是 Bash 的条件测试语法，比 [ ] 更强大
    while [[ $count -ne 0 ]] ; do
        # 每次循环暂停 2 秒，避免频繁查询
        sleep 2
        
        # 查询分布式账本，检查文件传输是否完成
        # curl：命令行 HTTP 客户端工具
        # grep "done"：在输出中搜索 "done" 字符串
        # | 管道符：将 curl 的输出传递给 grep 处理
        curl http://localhost:8080/api/ledger/tests/test | grep "done"
        
        # $? 保存上一个命令的退出状态码
        # 0 表示成功（找到 "done"），非 0 表示失败（未找到）
        rc=$?
        
        # 如果找到 "done"，说明接收端已完成接收
        if [[ $rc -eq 0 ]] ; then
            # 将计数器设为 1，下次循环会减到 0 并退出
            ((count = 1))
        fi
        
        # 计数器递减
        ((count = count - 1))
    done

    # 根据最终状态判断测试结果
    if [[ $rc -eq 0 ]] ; then
        # 测试成功
        echo "测试成功"
        # 等待 20 秒，确保所有资源正确释放
        sleep 20
        # 退出码 0 表示成功
        exit 0
    else
        # 测试失败
        echo "测试失败"
        # 退出码 1 表示失败
        exit 1
    fi
    
else
    # ==================== 接收端逻辑 ====================
    
    # 接收文件
    # --name "test"：指定要接收的文件名称标识
    # --path $PWD/test：指定接收文件的保存路径
    ./edgevpn file-receive --name "test" --path $PWD/test

    # 检查文件是否成功下载
    # -e 表示检查文件是否存在
    if [ ! -e $PWD/test ]; then
        echo "文件未下载"
        exit 1
    fi

    # 向分布式账本发送完成信号
    # -X PUT：指定 HTTP 请求方法为 PUT
    # 这会通知发送端文件已接收完成
    curl -X PUT http://localhost:8080/api/ledger/tests/test/done
    
    # 等待 80 秒，确保发送端有足够时间处理完成信号
    sleep 80
    
    # 读取接收到的文件内容
    # $( ) 是命令替换语法，将命令的输出赋值给变量
    t=$(cat $PWD/test)

    # 验证文件内容是否正确
    # 应该接收到内容为 "test" 的文件
    if [ $t != "test" ]; then
        echo "测试失败，返回内容为：$t"
        exit 1
    fi
fi


