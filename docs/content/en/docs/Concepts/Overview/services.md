---
title: "隧道连接"
linkTitle: "隧道"
weight: 1
description: >
  EdgeVPN 用于隧道 TCP 服务的网络服务
---

## 转发本地连接

EdgeVPN 也可以用于暴露本地（或远程）服务，而无需建立 VPN 和分配本地 tun/tap 设备，类似于 `ngrok`。

### 暴露服务

如果你习惯于本地 SSH 转发的工作方式（例如 `ssh -L 9090:something:remote <my_node>`），EdgeVPN 采用类似的方法。

服务是在主机上运行的通用 TCP 服务（也可以在网络外部）。例如，假设我们要暴露 LAN 内的 SSH 服务器。

要将服务暴露到你的 EdgeVPN 网络，则：

```bash
$ edgevpn service-add "MyCoolService" "127.0.0.1:22"
```

要访问该服务，EdgeVPN 将设置一个本地端口并绑定到它，它将通过 VPN 将流量隧道传输到该服务，例如，要在本地绑定到 `9090`：

```bash
$ edgevpn service-connect "MyCoolService" "127.0.0.1:9090"
```

在上面的示例中，在本地 SSH 连接到 `9090` 将转发到 `22`。
