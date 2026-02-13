---
title: "WebUI 和 API"
linkTitle: "WebUI 和 API"
weight: 1
description: >
  使用内置 API 查询网络状态并操作账本
---

API 内置了一个简单的 WebUI，用于显示网络信息。


要访问 Web 界面，在控制台中运行：

```bash
$ edgevpn api
```

使用 `EDGEVPNCONFIG` 或 `EDGEVPNTOKEN`。

仪表盘（深色模式）            |  仪表盘（浅色模式）
:-------------------------:|:-------------------------:
![Screenshot 2021-10-31 at 00-12-16 EdgeVPN - Machines index](https://user-images.githubusercontent.com/2420543/163020448-8e9238c1-3b6d-435d-9b25-7729d8779ebd.png) | ![Screenshot 2021-10-31 at 23-03-26 EdgeVPN - Machines index](https://user-images.githubusercontent.com/2420543/163020460-e18c07d7-8426-4992-aab3-0b2fd90279ae.png)

DNS            |  机器索引
:-------------------------:|:-------------------------:
![Screenshot 2021-10-31 at 23-03-44 EdgeVPN - Services index](https://user-images.githubusercontent.com/2420543/163020465-3d481da4-4912-445e-afc0-2614966dcadf.png) | ![Screenshot 2021-10-31 at 23-03-59 EdgeVPN - Files index](https://user-images.githubusercontent.com/2420543/163020462-7821a622-8c13-4971-8abe-9c5b6b491ae8.png)

服务            |  区块链索引
:-------------------------:|:-------------------------:
![Screenshot 2021-10-31 at 23-04-12 EdgeVPN - Users connected](https://user-images.githubusercontent.com/2420543/163021285-3c5a980d-2562-4c10-b266-7e99f19d8a87.png) | ![Screenshot 2021-10-31 at 23-04-20 EdgeVPN - Blockchain index](https://user-images.githubusercontent.com/2420543/163020457-77ef6e50-40a6-4e3b-83c4-a81db729bd7d.png)


在 API 模式下，EdgeVPN 将连接到网络，但不会路由任何数据包，也不会设置 VPN 接口。

默认情况下，edgevpn 将监听 `8080` 端口。有关可用选项，请参阅 `edgevpn api --help`

API 也可以与 VPN 一起启动，使用 `--api`。

## API 端点

### GET

#### `/api/users`

返回连接到区块链中服务的用户

#### `/api/services`

返回在区块链中运行的服务

#### `/api/dns`

返回在区块链中注册的域名

#### `/api/machines`

返回连接到 VPN 的机器

#### `/api/blockchain`

返回最新的可用区块链

#### `/api/ledger`

返回账本中的当前数据

#### `/api/ledger/:bucket`

返回 `:bucket` 内账本中的当前数据

#### `/api/ledger/:bucket/:key`

返回 `:bucket` 内指定 `:key` 处账本中的当前数据

#### `/api/peergate`

返回 peergater 状态

### PUT

#### `/api/ledger/:bucket/:key/:value`

将 `:value` 放入 `:bucket` 内指定 `:key` 处的账本中

#### `/api/peergate/:state`

启用/禁用 peergating：

```bash
# 启用
$ curl -X PUT 'http://localhost:8080/api/peergate/enable'
# 禁用
$ curl -X PUT 'http://localhost:8080/api/peergate/disable'
```

### POST

#### `/api/dns`

该端点接受以下形式的 JSON 载荷：

```json
{ "Regex": "<regex>", 
  "Records": { 
     "A": "2.2.2.2",
     "AAAA": "...",
  },
}
```

接受一个正则表达式和一组记录，并将它们注册到区块链。

账本中的 DNS 表将被嵌入式 DNS 服务器用于在本地处理请求。

例如，要创建新条目：

```bash
$ curl -X POST http://localhost:8080/api/dns --header "Content-Type: application/json" -d '{ "Regex": "foo.bar", "Records": { "A": "2.2.2.2" } }'
```

### DELETE

#### `/api/ledger/:bucket/:key`

删除账本内 `:bucket` 中的 `:key`

#### `/api/ledger/:bucket`

从账本中删除 `:bucket`

## 绑定到套接字

API 也可以绑定到套接字，例如：

```bash
$ edgevpn api --listen "unix://<path/to/socket>"
```

或者在运行 VPN 时：

```bash
$ edgevpn api --api-listen "unix://<path/to/socket>"
```
