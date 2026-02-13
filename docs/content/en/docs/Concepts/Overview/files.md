---
title: "发送和接收文件"
linkTitle: "文件传输"
weight: 20
date: 2017-01-05
description: >
  在 P2P 节点之间发送和接收文件
---

## 发送和接收文件

EdgeVPN 可以使用 `file-send` 和 `file-receive` 子命令在主机之间通过 P2P 发送和接收文件。

发送和接收文件与服务一样，不会建立 VPN 连接。

### 发送

```bash
$ edgevpn file-send --name unique-id --path /src/path
```

### 接收

```bash
$ edgevpn file-receive --name unique-id --path /dst/path
```
