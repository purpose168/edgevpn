---
title: "架构"
linkTitle: "架构"
weight: 2
description: >
  EdgeVPN 内部架构
resources:
- src: "**edgevpn_*.png"
---
 
## 简介

EdgeVPN 使用 [libp2p](https://github.com/libp2p/go-libp2p) 建立一个去中心化的、非对称加密的 gossip 网络，该网络在节点之间传播（对称加密的）区块链状态。

区块链是轻量级的，因为：
- 没有 PoW 机制
- 仅在内存中，没有 DAG、CARv2 或 GraphSync 协议 - 用途仅限于保存元数据，而不是真正的可寻址内容

EdgeVPN 使用区块链存储服务 UUID、文件 UUID、VPN 和其他元数据（如 DNS 记录、IP 等），并协调网络节点之间的事件。此外，它还用作保护机制：如果节点不属于区块链，它们就无法相互通信。

区块链是临时的且在内存中，可以选择存储在磁盘上。

每个节点都会持续广播其状态，直到它在区块链中协调一致。如果区块链从头开始，主机将重新宣布并尝试用它们的数据填充区块链。


- 简单（KISS）接口，用于显示区块链中的网络数据
- 使用 libp2p 在节点之间进行非对称 P2P 加密
- 从 OTP 密钥动态生成的 rendezvous 点
- 额外的 AES 对称加密。以防 rendezvous 点被泄露
- 区块链用作路由表的密封加密存储
- 连接是主机到主机创建的，并进行非对称加密

### 连接引导

网络使用 libp2p 引导，由 3 个阶段组成：

{{< imgproc edevpn_bootstrap.png Fit "1200x550" >}}
{{< /imgproc >}}

在第一阶段，节点通过 DHT 和自动通过 OTP 生成的 rendezvous 密钥相互发现。

一旦节点相互了解，就会建立一个 gossip 网络，节点通过 P2P 端到端加密通道交换区块链。区块链使用通过 OTP 轮换的对称密钥进行密封，该密钥在节点之间共享。

此时，在节点之间建立了区块链和 API，并可选择在 tun/tap 设备上启动 VPN 绑定。
