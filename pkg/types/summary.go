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

// Summary 网络摘要信息结构体
// 用于表示整个VPN网络的统计摘要
type Summary struct {
	Files        int    // 文件数量
	Machines     int    // 机器数量
	Users        int    // 用户数量
	Services     int    // 服务数量
	BlockChain   int    // 区块链区块数量
	OnChainNodes int    // 链上节点数量
	Peers        int    // 对等节点数量
	NodeID       string // 当前节点ID
}
