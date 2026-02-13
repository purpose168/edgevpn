<h1 align="center">
  <br>
	<img src="https://user-images.githubusercontent.com/2420543/144679248-1f6e4c10-a558-424c-b6f5-b3695269c906.png" width=128
         alt="logo"><br>
    EdgeVPN

<br>
</h1>

<h3 align="center">创建去中心化私有网络 </h3>
<p align="center">
  <a href="https://opensource.org/licenses/">
    <img src="https://img.shields.io/badge/licence-GPL3-brightgreen"
         alt="license">
  </a>
  <a href="https://github.com/purpose168/edgevpn/issues"><img src="https://img.shields.io/github/issues/purpose168/edgevpn"></a>
  <img src="https://img.shields.io/badge/made%20with-Go-blue">
  <img src="https://goreportcard.com/badge/github.com/purpose168/edgevpn" alt="go report card" />
</p>

<p align="center">
	 <br>
    完全去中心化。不可变。便携。易于使用的静态编译 VPN 和基于 p2p 的反向代理。<br>
    <b>VPN</b> -  <b>反向代理</b> - <b>通过 p2p 安全发送文件</b> -  <b>区块链</b>
</p>


EdgeVPN 使用 libp2p 构建可以通过共享密钥访问的私有去中心化网络。

它可以：

- **创建 VPN**：在 p2p 对等节点之间建立安全 VPN
  - 自动为节点分配 IP 地址
  - 内置微型 DNS 服务器用于解析内部/外部 IP
  - 创建可信区域以防止令牌泄露时的网络访问
  - 例如，[Kairos](https://github.com/kairos-io/kairos) CNCF 项目将其用作创建 Kubernetes 去中心化集群的层

- **作为反向代理**：像使用 `ngrok` 一样共享 TCP 服务。EdgeVPN 允许将 TCP 服务暴露给 p2p 网络节点而无需建立 VPN 连接：创建反向代理并将流量隧道化到 p2p 网络中。

- **通过 p2p 发送文件**：在节点之间通过 p2p 发送文件，无需建立 VPN 连接。

- **作为库使用**：轻松在您的 Go 代码中插入分布式 p2p 账本！例如，EdgeVPN 为 [LocalAI](https://github.com/mudler/LocalAI) 的 P2P 功能提供支持（您可以在[这里](https://localai.io/features/distribute/)了解更多）。

查看[文档](https://mudler.github.io/edgevpn)。

# :camera: 截图

仪表板（暗色模式）            |  仪表板（亮色模式）
:-------------------------:|:-------------------------:
![Screenshot 2021-10-31 at 00-12-16 EdgeVPN - Machines index](https://user-images.githubusercontent.com/2420543/163020448-8e9238c1-3b6d-435d-9b25-7729d8779ebd.png) | ![Screenshot 2021-10-31 at 23-03-26 EdgeVPN - Machines index](https://user-images.githubusercontent.com/2420543/163020460-e18c07d7-8426-4992-aab3-0b2fd90279ae.png)

DNS            |  机器索引
:-------------------------:|:-------------------------:
![Screenshot 2021-10-31 at 23-03-44 EdgeVPN - Services index](https://user-images.githubusercontent.com/2420543/163020465-3d481da4-4912-445e-afc0-2614966dcadf.png) | ![Screenshot 2021-10-31 at 23-03-59 EdgeVPN - Files index](https://user-images.githubusercontent.com/2420543/163020462-7821a622-8c13-4971-8abe-9c5b6b491ae8.png)

服务            |  区块链索引
:-------------------------:|:-------------------------:
![Screenshot 2021-10-31 at 23-04-12 EdgeVPN - Users connected](https://user-images.githubusercontent.com/2420543/163021285-3c5a980d-2562-4c10-b266-7e99f19d8a87.png) | ![Screenshot 2021-10-31 at 23-04-20 EdgeVPN - Blockchain index](https://user-images.githubusercontent.com/2420543/163020457-77ef6e50-40a6-4e3b-83c4-a81db729bd7d.png)


# :new: 图形界面

Linux 的桌面 GUI 应用程序（alpha 版本）可在[这里](https://github.com/purpose168/edgevpn-gui)获取

仪表板            |  连接索引
:-------------------------:|:-------------------------:
![edgevpn-gui-2](https://user-images.githubusercontent.com/2420543/147854909-a223a7c1-5caa-4e90-b0ac-0ae04dc0949d.png) | ![edgevpn-3](https://user-images.githubusercontent.com/2420543/147854904-09d96991-8752-421a-a301-8f0bdd9d5542.png)
![edgevpn-gui](https://user-images.githubusercontent.com/2420543/147854907-1e4a4715-3181-4dc2-8bc0-d052b3bf46d3.png) | 

# Kubernetes 

查看 [Kairos](https://github.com/kairos-io/kairos) 了解 EdgeVPN 在 Kubernetes 中的应用！

# :running: 安装

在[发布页面](https://github.com/purpose168/edgevpn/releases)下载预编译的静态版本。您可以将其安装到系统中或直接运行。

# :computer: 使用方法

EdgeVPN 通过生成令牌（或配置文件）来工作，这些令牌可以在不同的机器、主机或对等节点之间共享，以访问它们之间的去中心化安全网络。

每个令牌都是唯一的并标识网络，无需中央服务器设置或指定主机 IP。

要生成配置，运行：

```bash
# 生成新的配置文件，稍后用作 EDGEVPNCONFIG
$ edgevpn -g > config.yaml
```

或生成便携式令牌：

```bash
$ EDGEVPNTOKEN=$(edgevpn -g -b)
```

注意，令牌仅仅是 base64 编码的配置，所以这等同于：

```bash
$ EDGEVPNTOKEN=$(edgevpn -g | tee config.yaml | base64 -w0)
```

所有 edgevpn 命令都意味着您要么指定 `EDGEVPNTOKEN`（或作为参数的 `--token`），要么指定 `EDGEVPNCONFIG`，因为这是 `edgevpn` 在节点之间建立网络的方式。

配置文件是网络定义，允许您安全地连接到对等节点。

**警告** 暴露此文件或传递它等同于授予网络的完全控制权。

## :satellite: 作为 VPN

要启动 VPN，只需运行 `edgevpn` 而不带任何参数。

在多个主机上运行 edgevpn 的示例：

```bash
# 在节点 A 上
$ EDGEVPNTOKEN=.. edgevpn --address 10.1.0.11/24
# 在节点 B 上
$ EDGEVPNTOKEN=.. edgevpn --address 10.1.0.12/24
# 在节点 C 上 ...
$ EDGEVPNTOKEN=.. edgevpn --address 10.1.0.13/24
...
```

... 就是这样！`--address` 是每个节点的_虚拟_唯一 IP，实际上是节点在 VPN 中可访问的 IP。您可以自由地将 IP 分配给网络的节点，同时您可以使用 `IFACE`（或 `--interface`）覆盖默认的 `edgevpn0` 接口

*注意*：建立节点之间的连接可能需要时间。至少等待 5 分钟，具体取决于主机背后的网络。


# :question: 它适合我吗？

EdgeVPN 将 VPN 去中心化作为首要要求。

它的主要用途是边缘和低端设备，特别是用于开发。

去中心化方法有几个缺点：

- 底层网络是繁忙的。它使用 Gossip 协议来同步路由表和 p2p。每个区块链消息都会广播给所有对等节点，而流量仅流向主机。
- 可能不适合低延迟工作负载。

在生产网络中使用之前请记住这一点！

但它有一个强大的优点：它可以在 libp2p 工作的任何地方工作！

# :question: 为什么？ 

首先，这是我第一次尝试 libp2p。其次，我一直想要一个更"开放"的 `ngrok` 替代品，但我总是更喜欢维护"更少的基础设施"。这就是为什么在 `libp2p` 之上构建这样的东西是有意义的。

# :warning: 警告！

我不是安全专家，这个软件没有经过完整的安全审计，所以不要依赖它处理敏感流量，也不要用于生产环境！我主要是为了在尝试 libp2p 时娱乐而制作的。

## 示例用例：网络去中心化的 [k3s](https://github.com/k3s-io/k3s) 测试集群

让我们看一个实际示例，您正在为 Kubernetes 开发某些东西，并且想尝试多节点设置，但您只有位于 NAT 后面的机器（可惜！），您真的很想利用硬件。

如果您真的对网络性能不感兴趣（同样，这仅用于开发目的！），那么您可以这样使用 `edgevpn` + [k3s](https://github.com/k3s-io/k3s)：

1) 生成 edgevpn 配置：`edgevpn -g > vpn.yaml`
2) 启动 VPN：

   在节点 A 上：`sudo IFACE=edgevpn0 ADDRESS=10.1.0.3/24 EDGEVPNCONFIG=vpn.yml edgevpn`
   
   在节点 B 上：`sudo IFACE=edgevpn0 ADDRESS=10.1.0.4/24 EDGEVPNCONFIG=vpm.yml edgevpn`
3) 启动 k3s：
 
   在节点 A 上：`k3s server --flannel-iface=edgevpn0`
   
   在节点 B 上：`K3S_URL=https://10.1.0.3:6443 K3S_TOKEN=xx k3s agent --flannel-iface=edgevpn0 --node-ip 10.1.0.4`

我们在这里使用了 flannel，但其他 CNI 也应该可以工作。


# :notebook: 作为库

EdgeVPN 可以作为库使用。它非常便携并提供功能接口。

要从令牌加入网络中的节点，而不启动 VPN：

```golang

import (
    node "github.com/purpose168/edgevpn/pkg/node"
)

e := node.New(
    node.Logger(l),
    node.LogLevel(log.LevelInfo),
    node.MaxMessageSize(2 << 20),
    node.FromBase64( mDNSEnabled, DHTEnabled, token ),
    // ....
  )

e.Start(ctx)

```

或启动 VPN：

```golang

import (
    vpn "github.com/purpose168/edgevpn/pkg/vpn"
    node "github.com/purpose168/edgevpn/pkg/node"
)

opts, err := vpn.Register(vpnOpts...)
if err != nil {
	return err
}

e := edgevpn.New(append(o, opts...)...)

e.Start(ctx)
```

# 🧑‍💻 使用 EdgeVPN 的项目

- [Kairos](https://github.com/kairos-io/kairos) - 使用 EdgeVPN 网络自动创建 K3s Kubernetes 集群


# 🐜 贡献

您可以通过以下方式改进此项目：

- 报告错误
- 修复问题
- 请求功能
- 提问（只需开一个 issue）

以及此处未提及的任何其他方式。

# :notebook: 致谢

- 棒极了的 [libp2p](https://github.com/libp2p) 库
- [https://github.com/songgao/water](https://github.com/songgao/water) 用于 Go 中的 tun/tap 设备
- [Room 示例](https://github.com/libp2p/go-libp2p/tree/master/examples/chat-with-rendezvous)（无耻地复制了部分内容）
- Logo 最初由 [Uniconlabs](https://www.flaticon.com/authors/uniconlabs) 从 [www.flaticon.com](https://www.flaticon.com/) 制作，由我修改

# :notebook: 故障排除

如果在引导过程中您看到如下消息：

```
edgevpn[3679]:             * [/ip4/104.131.131.82/tcp/4001] failed to negotiate stream multiplexer: context deadline exceeded     
```

或

```
edgevpn[9971]: 2021/12/16 20:56:34 failed to sufficiently increase receive buffer size (was: 208 kiB, wanted: 2048 kiB, got: 416 kiB). See https://github.com/lucas-clemente/quic-go/wiki/UDP-Receive-Buffer-Size for details.
```

或通常遇到网络性能不佳，建议通过运行以下命令增加最大缓冲区大小：

```
sysctl -w net.core.rmem_max=2500000
```

# :notebook: 待办事项

- [x] VPN
- [x] 通过 p2p 发送和接收文件
- [x] 通过 p2p 隧道暴露远程/本地服务
- [x] 在区块链上存储任意数据
- [x] 允许在磁盘上持久化区块链

# :notebook: 许可证

Apache License v2.

```
edgevpn  Copyright (C) 2021 Ettore Di Giacinto
This program comes with ABSOLUTELY NO WARRANTY.
This is free software, and you are welcome to redistribute it
under certain conditions.
```
