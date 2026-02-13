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

package types

// Machine 机器信息结构体
// 用于表示网络中的机器节点信息
type Machine struct {
	PeerID   string // 对等节点ID
	Hostname string // 主机名
	OS       string // 操作系统类型
	Arch     string // 系统架构
	Address  string // IP地址
	Version  string // 软件版本
}
