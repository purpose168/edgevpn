---
title: "Peerguardian"
linkTitle: "Peerguardian"
weight: 25
date: 2022-01-05
description: >
  在令牌泄露时防止对网络的未授权访问
math: false
---

{{% pageinfo color="warning"%}}
实验性功能！
{{% /pageinfo %}}

## Peerguardian

PeerGuardian 是一种机制，用于在令牌泄露时防止对网络的未授权访问或撤销网络访问。

要启用它，启动 edgevpn 节点时添加 `--peerguradian` 标志。

```bash
edgevpn --peerguardian
```

要启用 peer gating，还需要指定 `--peergate`。

Peerguardian 和 peergating 有多个选项：

```
   --peerguard                                   启用 peerguard。（实验性）[$PEERGUARD]
   --peergate                                    启用 peergating。（实验性）[$PEERGATE]
   --peergate-autoclean                          启用 peergating 自动清理。（实验性）[$PEERGATE_AUTOCLEAN]
   --peergate-relaxed                            启用 peergating 宽松模式。（实验性）[$PEERGATE_RELAXED]
   --peergate-auth value                         Peergate 认证 [$PEERGATE_AUTH]
   --peergate-interval value                     Peergater 间隔时间（默认：120）[$EDGEVPNPEERGATEINTERVAL]
```

当启用 PeerGuardian 和 Peergater 时，VPN 节点将只接受来自授权节点的区块。

Peerguardian 是可扩展的，支持不同的认证机制，我们将在下面介绍具体的实现。

## ECDSA 认证

ECDSA 认证机制用于使用 ECDSA 密钥验证区块链中的节点。

要生成新的 ECDSA 密钥对，使用 `edgevpn peergater ecdsa-genkey`：

```bash
$ edgevpn peergater ecdsa-genkey
Private key: LS0tLS1CRUdJTiBFQyBQUklWQVRFIEtFWS0tLS0tCk1JSGNBZ0VCQkVJQkhUZnRSTVZSRmlvaWZrdllhZEE2NXVRQXlSZTJSZHM0MW1UTGZlNlRIT3FBTTdkZW9sak0KZXVPbTk2V0hacEpzNlJiVU1tL3BCWnZZcElSZ0UwZDJjdUdnQndZRks0RUVBQ09oZ1lrRGdZWUFCQUdVWStMNQptUzcvVWVoSjg0b3JieGo3ZmZUMHBYZ09MSzNZWEZLMWVrSTlEWnR6YnZWOUdwMHl6OTB3aVZxajdpMDFVRnhVCnRKbU1lWURIRzBTQkNuVWpDZ0FGT3ByUURpTXBFR2xYTmZ4LzIvdEVySDIzZDNwSytraFdJbUIza01QL2tRNEIKZzJmYnk2cXJpY1dHd3B4TXBXNWxKZVZXUGlkeWJmMSs0cVhPTWdQbmRnPT0KLS0tLS1FTkQgRUMgUFJJVkFURSBLRVktLS0tLQo=
Public key: LS0tLS1CRUdJTiBFQyBQVUJMSUMgS0VZLS0tLS0KTUlHYk1CQUdCeXFHU000OUFnRUdCU3VCQkFBakE0R0dBQVFCbEdQaStaa3UvMUhvU2ZPS0syOFkrMzMwOUtWNApEaXl0MkZ4U3RYcENQUTJiYzI3MWZScWRNcy9kTUlsYW8rNHROVkJjVkxTWmpIbUF4eHRFZ1FwMUl3b0FCVHFhCjBBNGpLUkJwVnpYOGY5djdSS3g5dDNkNlN2cElWaUpnZDVERC81RU9BWU5uMjh1cXE0bkZoc0tjVEtWdVpTWGwKVmo0bmNtMzlmdUtsempJRDUzWT0KLS0tLS1FTkQgRUMgUFVCTElDIEtFWS0tLS0tCg==
```

例如，要添加 ECDSA 公钥，从已被 PeerGuardian 信任的节点使用 API：

```bash
$ curl -X PUT 'http://localhost:8080/api/ledger/trustzoneAuth/ecdsa_1/LS0tLS1CRUdJTiBFQyBQVUJMSUMgS0VZLS0tLS0KTUlHYk1CQUdCeXFHU000OUFnRUdCU3VCQkFBakE0R0dBQVFBL09TTjhsUU9Wa3FHOHNHbGJiellWamZkdVVvUAplMEpsWUVzOFAyU3o1TDlzVUtDYi9kQWkrVFVONXU0ZVk2REpGeU50dWZjK2p0THNVTTlPb0xXVnBXb0E0eEVDCk9VdDFmRVNaRzUxckc4MEdFVjBuQTlBRGFvOW1XK3p4dmkvQnd0ZFVvSTNjTDB0VTdlUGEvSGM4Z1FLMmVOdE0KeDdBSmNYcWpPNXZXWGxZZ2NkOD0KLS0tLS1FTkQgRUMgUFVCTElDIEtFWS0tLS0tCg=='
```

现在可以在启动新节点时使用私钥：

```bash
PEERGATE_AUTH="{ 'ecdsa' : { 'private_key': 'LS0tLS1CRUdJTiBFQyBQUklWQVRFIEtFWS0tLS0tCk1JSGNBZ0VCQkVJQkhUZnRSTVZSRmlvaWZrdllhZEE2NXVRQXlSZTJSZHM0MW1UTGZlNlRIT3FBTTdkZW9sak0KZXVPbTk2V0hacEpzNlJiVU1tL3BCWnZZcElSZ0UwZDJjdUdnQndZRks0RUVBQ09oZ1lrRGdZWUFCQUdVWStMNQptUzcvVWVoSjg0b3JieGo3ZmZUMHBYZ09MSzNZWEZLMWVrSTlEWnR6YnZWOUdwMHl6OTB3aVZxajdpMDFVRnhVCnRKbU1lWURIRzBTQkNuVWpDZ0FGT3ByUURpTXBFR2xYTmZ4LzIvdEVySDIzZDNwSytraFdJbUIza01QL2tRNEIKZzJmYnk2cXJpY1dHd3B4TXBXNWxKZVZXUGlkeWJmMSs0cVhPTWdQbmRnPT0KLS0tLS1FTkQgRUMgUFJJVkFURSBLRVktLS0tLQo=' } }"
$ edgevpn --peerguardian --peergate
```

## 在运行时启用/禁用 peergating

可以通过 API 在运行时禁用 peergating：

### 查询状态

```bash
$ curl -X GET 'http://localhost:8080/api/peergate'
```

### 启用 peergating
```bash
$ curl -X PUT 'http://localhost:8080/api/peergate/enable'
```

### 禁用 peergating
```bash
$ curl -X PUT 'http://localhost:8080/api/peergate/disable'
```

## 启动新网络

要初始化新的可信网络，使用 `--peergate-relaxed` 启动节点并添加必要的认证密钥：

```bash
$ edgevpn --peerguardian --peergate --peergate-relaxed
$ curl -X PUT 'http://localhost:8080/api/ledger/trustzoneAuth/keytype_1/XXX'
```

{{% alert title="注意" %}}
强烈建议在使用 PeerGuardian 时使用区块链的本地存储。这样节点可以在本地持久化认证密钥，你可以避免使用 `--peergate-relaxed` 启动节点
{{% /alert %}}
