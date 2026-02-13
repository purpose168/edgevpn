/*
Copyright © 2021-2022 Ettore Di Giacinto <mudler@mocaccino.org>
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package protocol

import (
	p2pprotocol "github.com/libp2p/go-libp2p/core/protocol"
)

// 协议ID常量定义
const (
	EdgeVPN         Protocol = "/edgevpn/0.1"       // EdgeVPN主协议
	ServiceProtocol Protocol = "/edgevpn/service/0.1" // 服务协议
	FileProtocol    Protocol = "/edgevpn/file/0.1"    // 文件协议
	EgressProtocol  Protocol = "/edgevpn/egress/0.1"  // 出口协议
)

// 账本键常量定义
const (
	FilesLedgerKey    = "files"        // 文件账本键
	MachinesLedgerKey = "machines"     // 机器账本键
	ServicesLedgerKey = "services"     // 服务账本键
	UsersLedgerKey    = "users"        // 用户账本键
	HealthCheckKey    = "healthcheck"  // 健康检查键
	DNSKey            = "dns"          // DNS键
	EgressService     = "egress"       // 出口服务键
	TrustZoneKey      = "trustzone"    // 信任区域键
	TrustZoneAuthKey  = "trustzoneAuth" // 信任区域认证键
)

// Protocol 协议类型定义
type Protocol string

// ID 返回libp2p协议ID
func (p Protocol) ID() p2pprotocol.ID {
	return p2pprotocol.ID(string(p))
}
