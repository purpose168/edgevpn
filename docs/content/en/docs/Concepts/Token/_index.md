---
title: "令牌"
linkTitle: "令牌"
weight: 3
description: >
  EdgeVPN 网络令牌
math: false
---

网络令牌代表 EdgeVPN 尝试在节点之间建立连接的网络。

令牌是通过将网络配置编码为 base64 创建的。

## 生成令牌

要生成网络令牌，在控制台中运行：

```
edgevpn -b -g
```

这将在屏幕上打印出一个 base64 令牌，可以将其共享给你希望加入同一网络的节点。

## 生成配置文件

EdgeVPN 可以读取令牌和网络配置文件。

要生成配置文件，在控制台中运行：

```
edgevpn -g
```

要将配置转换为令牌，必须编码为 base64：

```
TOKEN=$(edgevpn -g | base64 -w0)
```

这相当于运行 `edgevpn -g -b`。

## 配置文件剖析

典型的配置文件如下所示：

```yaml
otp:
  dht:
    interval: 9000
    key: LHKNKT6YZYQGGY3JANGXMLJTHRH7SW3C
    length: 32
  crypto:
    interval: 9000
    key: SGIB6NYJMSRJF2AJDGUI2NDB5LBVCPLS
    length: 32
room: ubONSBFkdWbzkSBTglFzOhWvczTBQJOR
rendezvous: exoHOajMYMSPrHhevAEEjnCHLssFfzfT
mdns: VoZfePlTchbSrdmivaqaOyQyEnTMlugi
max_message_size: 20971520
```

所有值都可以根据你的需求进行调整。

EdgeVPN 使用 OTP 机制来解密节点之间的区块链消息并从 DHT 发现节点，这是为了防止暴力攻击并避免恶意行为者监听协议。
有关更多信息，请参阅[架构部分]()。

- OTP 密钥（`otp.crypto.key`）轮换用于编码/解码区块链消息的密码密钥。可以为 DHT 和区块链消息设置轮换间隔。长度是密封器用于解密/加密消息的密码密钥长度（默认为 AES-256）。
- DHT OTP 密钥（`otp.dht.key`）轮换在 DHT 节点发现期间使用的发现密钥。在定义的间隔使用 OTP 生成并使用密钥，以干扰潜在的监听者。
- `room` 是所有节点将订阅的唯一 ID。它是自动生成的
- 可以通过注释掉 `otp` 块来选择性地禁用 OTP 机制。在这种情况下，静态 DHT rendezvous 将是 `rendezvous`
- `mdns` 发现没有任何 OTP 轮换，因此必须提供唯一标识符。
- 这里可以使用 `max_message_size`（以字节为单位）定义区块链消息接受的最大消息大小
