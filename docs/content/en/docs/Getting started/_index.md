---
title: "入门指南"
linkTitle: "入门指南"
weight: 1
description: >
  EdgeVPN 的第一步
---

## 获取 EdgeVPN  

先决条件：无依赖。EdgeVPN 发布版本是静态编译的。

### 从发布版本获取

只需从 [GitHub 发布页面](https://github.com/purpose168/edgevpn/releases)获取发布版本。二进制文件是静态编译的。

### 通过 Homebrew 在 MacOS 上安装

如果你在 MacOS 上使用 homebrew，可以使用 [edgevpn formula](https://formulae.brew.sh/formula/edgevpn)

```
brew install edgevpn
```


### 从源代码构建 EdgeVPN

要求：

- 系统中安装了 [Golang](https://golang.org/)。
- make

```bash
$> git clone https://github.com/purpose168/edgevpn
$> cd edgevpn
$> go build
```

### 使用 Docker Compose

使用 docker 仍然是实验性的，因为设置可能会有很大差异。
为了方便起见，提供了一个示例 [docker-compose.yml](https://github.com/purpose168/edgevpn/blob/master/docker-compose.yml) 文件，但你可能需要编辑它。

```bash
$> git clone https://github.com/purpose168/edgevpn
$> cd edgevpn
$> sudo docker compose up --detach
```

## 创建你的第一个 VPN

现在让我们创建第一个 VPN 并启动它：

```bash
$> EDGEVPNTOKEN=$(edgevpn -b -g)
$> edgevpn --dhcp --api
```

就是这样！

你现在可以在 [http://localhost:8080](http://localhost:8080) 访问 Web 界面。

要将新节点加入网络，只需复制 `EDGEVPNTOKEN` 并在其他节点上使用它来启动 edgevpn：

```bash
$> EDGEVPNTOKEN=<之前生成的令牌> edgevpn --dhcp
```
