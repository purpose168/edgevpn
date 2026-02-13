---
title: "DNS"
linkTitle: "DNS"
weight: 20
date: 2017-01-05
description: >
  嵌入式 DNS 服务器文档
math: false
---

{{% pageinfo color="warning"%}}
实验性功能！
{{% /pageinfo %}}

## DNS 服务器

DNS 服务器可用但默认禁用。

DNS 服务器将使用区块链作为记录来解析 DNS 查询，并默认转发未知域名。

可以通过使用 `--dns` 指定监听地址来启用。例如，要在本地绑定到默认的 `53` 端口，在控制台中运行：

```bash
edgevpn --dns "127.0.0.1:53"
```

要关闭 DNS 转发，指定 `--dns-forwarder=false`。可以选择使用 `--dns-forward-server` 多次指定 DNS 服务器列表。

dns 子命令有多个选项：

```
   --dns value                             DNS 监听地址。留空以禁用 DNS 服务器 [$DNSADDRESS]
   --dns-forwarder                         启用 DNS 转发 [$DNSFORWARD]                 
   --dns-cache-size value                  DNS LRU 缓存大小（默认：200）[$DNSCACHESIZE]                  
   --dns-forward-server value              DNS 转发服务器列表（默认："8.8.8.8:53", "1.1.1.1:53"）[$DNSFORWARDSERVER]
```

VPN 的节点可以启动本地 DNS 服务器，该服务器将解析存储在链中的路由。

例如，要添加 DNS 记录，使用 API：

```bash
$ curl -X POST http://localhost:8080/api/dns --header "Content-Type: application/json" -d '{ "Regex": "foo.bar", "Records": { "A": "2.2.2.2" } }'
```

`/api/dns` 路由接受以下形式的 `POST` 请求作为 `JSON`：

```json
{ "Regex": "<regex>", 
  "Records": { 
     "A": "2.2.2.2",
     "AAAA": "...",
  },
}
```

注意，`Regex` 接受正则表达式，将匹配接收到的 DNS 请求并解析为指定的条目。
