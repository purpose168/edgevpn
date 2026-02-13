#!/bin/bash
# =============================================================================
# EdgeVPN 服务测试脚本
# 用途：测试EdgeVPN的服务暴露和连接功能
# 作者：purpose168@outlook.com
# 日期：2026-02-13
# =============================================================================

# 启动edgevpn API服务（后台运行）
# & 符号表示在后台运行进程，不阻塞当前脚本执行
./edgevpn api &

# 根据第一个命令行参数判断测试模式
# $1 表示脚本的第一个参数
if [ $1 == "expose" ]; then
    # ========================================================================
    # 模式1：服务暴露测试（expose模式）
    # 此模式用于测试服务的暴露功能
    # ========================================================================
    
    # 添加一个测试服务，将testservice暴露在127.0.0.1:8080端口
    # service-add命令用于注册新服务
    ./edgevpn service-add testservice 127.0.0.1:8080 &
    
    # 初始化计数器为240秒（最多等待480秒，因为每次sleep 2秒）
    # (( )) 是bash的算术运算语法
    ((count = 240))                        
    
    # 循环等待服务就绪
    # while循环会持续执行，直到count为0或服务成功响应
    while [[ $count -ne 0 ]] ; do
        # 每次循环等待2秒
        sleep 2
        
        # 使用curl访问服务账本API，检查服务是否已注册
        # grep命令用于在输出中搜索"doneservice"字符串
        # | 是管道符，将curl的输出传递给grep
        curl http://localhost:8080/api/ledger/tests/services | grep "doneservice"
        
        # $? 保存上一个命令的退出状态码
        # 0 表示成功（grep找到了匹配项）
        # 非0 表示失败（grep没有找到匹配项）
        rc=$?
        
        # 如果grep成功找到匹配项，设置count为1以便退出循环
        if [[ $rc -eq 0 ]] ; then
            ((count = 1))
        fi
        
        # 计数器递减
        ((count = count - 1))
    done
    
    # 检查最终测试结果
    if [[ $rc -eq 0 ]] ; then
        # 测试成功，输出成功信息
        # "Alright" 表示测试通过
        echo "测试成功"
        
        # 等待20秒后退出
        sleep 20
        # exit 0 表示脚本成功退出
        exit 0
    else
        # 测试失败，输出失败信息
        echo "测试失败"
        # exit 1 表示脚本异常退出
        exit 1
    fi
    
else
    # ========================================================================
    # 模式2：服务连接测试（connect模式）
    # 此模式用于测试服务的连接功能
    # ========================================================================
    
    # 连接到testservice服务，将本地9090端口映射到远程服务
    # service-connect命令用于建立到远程服务的连接
    # :9090 表示监听本地所有网络接口的9090端口
    ./edgevpn service-connect testservice :9090 &
    
    # 初始化计数器为240秒（最多等待480秒）
    ((count = 240))                        
    
    # 循环等待服务连接建立并响应
    while [[ $count -ne 0 ]] ; do
        # 每次循环等待2秒
        sleep 2
        
        # 使用curl访问本地9090端口，检查是否能获取到EdgeVPN响应
        # 这验证了服务连接是否正常工作
        curl http://localhost:9090/ | grep "EdgeVPN"
        
        # 获取curl和grep组合命令的退出状态
        rc=$?
        
        # 如果成功找到"EdgeVPN"字符串，设置count为1以便退出循环
        if [[ $rc -eq 0 ]] ; then
            ((count = 1))
        fi
        
        # 计数器递减
        ((count = count - 1))
    done
    
    # 检查最终测试结果
    if [[ $rc -eq 0 ]] ; then
        # 测试成功
        echo "测试成功"
        
        # 向服务账本API发送PUT请求，标记服务测试完成
        # 这会通知其他节点（如expose模式的节点）服务已经成功连接
        # -X PUT 指定HTTP请求方法为PUT
        curl -X PUT http://localhost:8080/api/ledger/tests/services/doneservice
        
        # 等待80秒后退出
        # 较长的等待时间可能是为了确保其他测试节点有足够时间完成
        sleep 80
        exit 0
    else
        # 测试失败
        echo "测试失败"
        exit 1
    fi
fi
