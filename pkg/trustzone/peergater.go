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

package trustzone

import (
	"context"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/mudler/edgevpn/pkg/blockchain"
	"github.com/mudler/edgevpn/pkg/node"
	"github.com/mudler/edgevpn/pkg/protocol"
)

// PeerGater 对等节点门控器，用于控制对等节点的访问权限
type PeerGater struct {
	sync.Mutex
	trustDB          []peer.ID  // 信任数据库，存储已认证的对等节点ID
	enabled, relaxed bool       // enabled表示是否启用，relaxed表示是否宽松模式
}

// NewPeerGater 返回新的对等节点门控器
// 在宽松模式下，只有当trustDB包含认证数据时才会进行门控
// 参数 relaxed 表示是否启用宽松模式
func NewPeerGater(relaxed bool) *PeerGater {
	return &PeerGater{enabled: true, relaxed: relaxed}
}

// Enabled 返回对等节点门控器是否启用
func (pg *PeerGater) Enabled() bool {
	pg.Lock()
	defer pg.Unlock()
	return pg.enabled
}

// Disable 关闭对等节点门控机制
func (pg *PeerGater) Disable() {
	pg.Lock()
	defer pg.Unlock()
	pg.enabled = false
}

// Enable 开启对等节点门控机制
func (pg *PeerGater) Enable() {
	pg.Lock()
	defer pg.Unlock()
	pg.enabled = true
}

// Implements peergating interface
// resolves to peers in the trustDB. if peer is absent will return true
// Gate 实现对等节点门控接口
// 解析trustDB中的对等节点。如果对等节点不存在则返回true（表示需要门控）
// 参数 n 为节点实例，p 为要检查的对等节点ID
func (pg *PeerGater) Gate(n *node.Node, p peer.ID) bool {
	pg.Lock()
	defer pg.Unlock()
	// 如果未启用，不进行门控
	if !pg.enabled {
		return false
	}

	// 宽松模式下，如果信任数据库为空，不进行门控
	if pg.relaxed && len(pg.trustDB) == 0 {
		return false
	}

	// 检查对等节点是否在信任数据库中
	for _, pp := range pg.trustDB {
		if pp == p {
			return false
		}
	}

	return true
}

// UpdaterService 是负责从账本状态同步回trustDB的服务。
// 它是一个网络服务，检索信任区域中列出的发送者ID，
// 并将其填充到用于门控区块链消息的trustDB中
// 参数 duration 为更新间隔时间
func (pg *PeerGater) UpdaterService(duration time.Duration) node.NetworkService {
	return func(ctx context.Context, c node.Config, n *node.Node, b *blockchain.Ledger) error {
		b.Announce(ctx, duration, func() {
			db := []peer.ID{}
			// 获取信任区域数据
			tz, found := b.CurrentData()[protocol.TrustZoneKey]
			if found {
				// 将信任区域中的对等节点ID添加到数据库
				for k, _ := range tz {
					db = append(db, peer.ID(k))
				}
			}
			// 更新信任数据库
			pg.Lock()
			pg.trustDB = db
			pg.Unlock()
		})

		return nil
	}
}
