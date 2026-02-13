---
title: "CLI"
linkTitle: "CLI"
weight: 1
description: >
  命令行接口
---


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

... 就是这样！`--address` 是每个节点的_虚拟_唯一 IP，实际上是该节点在 VPN 中可访问的 IP。你可以自由地为网络节点分配 IP，同时可以使用 `IFACE`（或 `--interface`）覆盖默认的 `edgevpn0` 接口。

*注意*：建立节点之间的连接可能需要一些时间。请至少等待 5 分钟，具体取决于主机背后的网络情况。

VPN 支持多个选项，下面你将找到最重要功能的参考：


## 生成网络令牌

EdgeVPN 通过生成令牌（或网络配置文件）来工作，这些令牌在不同机器之间共享。

每个令牌都是唯一的，并标识网络本身：没有中央服务器设置，配置文件中也没有指定 IP 地址。

要生成新的网络令牌，只需运行 `edgevpn -g -b`：

```bash
$ edgevpn -g -b
b3RwOgogIGRodDoKICAgIGludGVydmFsOiA5MDAwCiAgICBrZXk6IDRPNk5aUUMyTzVRNzdKRlJJT1BCWDVWRUkzRUlKSFdECiAgICBsZW5ndGg6IDMyCiAgY3J5cHRvOgogICAgaW50ZXJ2YWw6IDkwMDAKICAgIGtleTogN1hTUUNZN0NaT0haVkxQR0VWTVFRTFZTWE5ORzNOUUgKICAgIGxlbmd0aDogMzIKcm9vbTogWUhmWXlkSUpJRlBieGZDbklLVlNmcGxFa3BhVFFzUk0KcmVuZGV6dm91czoga1hxc2VEcnNqbmFEbFJsclJCU2R0UHZGV0RPZGpXd0cKbWRuczogZ0NzelJqZk5XZEFPdHhubm1mZ3RlSWx6Zk1BRHRiZGEKbWF4X21lc3NhZ2Vfc2l6ZTogMjA5NzE1MjAK
```

为了连接并在节点之间建立网络连接，后续与 edgevpn 的所有交互都需要指定网络令牌。

例如，要在 API 模式下启动 `edgevpn`：

```bash
$ edgevpn api --token <token> # 或者使用 $EDGEVPNTOKEN
 INFO           edgevpn  Copyright (C) 2021-2022 Ettore Di Giacinto
       This program comes with ABSOLUTELY NO WARRANTY.
       This is free software, and you are welcome to redistribute it
       under certain conditions.
 INFO  Version: v0.8.4 commit:
 INFO   Starting EdgeVPN network
 INFO   Node ID: 12D3KooWRW4RXSMAh7CTRsTjX7iEjU6DEU8QKJZvFjSosv7zCCeZ
 INFO   Node Addresses: [/ip6/::1/tcp/38637 /ip4/192.168.1.234/tcp/41607 /ip4/127.0.0.1/tcp/41607]
 INFO   Bootstrapping DHT
⇨ http server started on [::]:8080
```

或者，可以使用 `--config` 或 `EDGEVPNCONFIG` 指定网络配置文件。

由于令牌是 base64 编码的网络配置文件，因此使用令牌或配置是等效的：

```bash
$ EDGEVPNTOKEN=$(edgevpn -g | tee config.yaml | base64 -w0)
```

## API

在 VPN 模式下启动时，也可以通过指定 `--api` 同时在 API 模式下启动。

## DHCP

注意：实验性功能！

从版本 `0.8.1` 开始，提供自动 IP 协商功能。

可以使用 `--dhcp` 启用 DHCP，并且可以省略 `--address`。如果使用 `--address` 指定了 IP，它将成为默认 IP。

## IPv6（实验性）

注意：非常实验性的功能！高度不稳定！

仅使用静态地址提供对 IPv6 的初步支持。目前每个接口仅支持一个地址，不支持双栈。
有关更多信息，请查看 [issue #15](https://github.com/mudler/edgevpn/issues/15)

可以使用 `--address fd:ed4e::<IP>/64` 和 `--mtu >1280` 启用 IPv6。
