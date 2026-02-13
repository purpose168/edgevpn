
---
title: "文档"
linkTitle: "文档"
weight: 20
menu:
  main:
    weight: 20
---


EdgeVPN 使用 libp2p 构建可以通过共享密钥访问的私有去中心化网络。

它可以：

- **创建 VPN** :  
  - P2P 节点之间的安全 VPN
  - 自动为节点分配 IP
  - 嵌入式微型 DNS 服务器，用于解析内部/外部 IP

- **充当反向代理**
  - 像 `ngrok` 一样将 TCP 服务共享到 P2P 网络节点，而无需建立 VPN 连接

- **通过 P2P 发送文件**
  - 在节点之间通过 P2P 发送文件，而无需建立 VPN 连接。

- **作为库使用**
  - 轻松在你的 Go 代码中插入分布式 P2P 账本！

查看下面的文档以获取更多示例和参考，请查看我们的[入门指南]({{< relref "/docs">}}/getting-started)、[CLI 接口]({{< relref "/docs">}}/getting-started/cli)、[GUI 桌面应用]({{< relref "/docs">}}/getting-started/gui)以及嵌入式 [WebUI/API]({{< relref "/docs">}}/getting-started/api/)。


| [WebUI]({{< relref "/docs">}}/getting-started/api)            | [桌面应用](https://github.com/mudler/edgevpn-gui)                                          |
| ------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------ |
| ![img](https://user-images.githubusercontent.com/2420543/163020448-8e9238c1-3b6d-435d-9b25-7729d8779ebd.png) | ![](https://user-images.githubusercontent.com/2420543/147854909-a223a7c1-5caa-4e90-b0ac-0ae04dc0949d.png) |
