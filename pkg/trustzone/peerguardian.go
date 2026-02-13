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
	"time"

	"github.com/ipfs/go-log"
	"github.com/purpose168/edgevpn/pkg/blockchain"
	"github.com/purpose168/edgevpn/pkg/hub"
	"github.com/purpose168/edgevpn/pkg/node"
	"github.com/purpose168/edgevpn/pkg/protocol"
)

// PeerGuardian 为区块链数据中的对等节点提供认证功能
type PeerGuardian struct {
	authProviders []AuthProvider     // 认证提供者列表
	logger        log.StandardLogger // 日志记录器
}

// NewPeerGuardian 创建新的对等节点守护者实例
// 参数 logger 为日志记录器，authProviders 为认证提供者列表
func NewPeerGuardian(logger log.StandardLogger, authProviders ...AuthProvider) *PeerGuardian {
	return &PeerGuardian{
		authProviders: authProviders,
		logger:        logger,
	}
}

// ReceiveMessage 是公共通道的通用处理器，用于提供认证功能。
// 我们在这里接收消息，并根据两个标准进行选择：
//   - 用于生成认证机制挑战的消息。
//     认证机制应从专门用于手动添加哈希值的TZ区域获取用户认证数据
//   - 对此类挑战的回答消息，意味着应该将sender.ID添加到信任区域
func (pg *PeerGuardian) ReceiveMessage(l *blockchain.Ledger, m *hub.Message, c chan *hub.Message) error {
	pg.logger.Debug("对等节点守护者收到来自", m.SenderID, "的消息")

	for _, a := range pg.authProviders {

		_, exists := l.GetKey(protocol.TrustZoneKey, m.SenderID)
		trustAuth := l.CurrentData()[protocol.TrustZoneAuthKey]
		if !exists && a.Authenticate(m, c, trustAuth) {
			// 尝试认证
			// 注意，我们可能不在这里的TZ中，因为我们无法检查（手头缺少节点信息）
			// 无论如何，节点会忽略消息，而我们触发Authenticate对于两步（或更多）认证器是有用的
			l.Persist(context.Background(), 5*time.Second, 120*time.Second, protocol.TrustZoneKey, m.SenderID, "")
			return nil
		}
	}

	return nil
}

// Challenger 是一个网络服务，应该使用所有启用的认证器发送挑战，直到我们进入TZ
// 注意这可能永远不会发生，因为节点可能没有满足的认证机制
func (pg *PeerGuardian) Challenger(duration time.Duration, autocleanup bool) node.NetworkService {
	return func(ctx context.Context, c node.Config, n *node.Node, b *blockchain.Ledger) error {
		b.Announce(ctx, duration, func() {
			trustAuth := b.CurrentData()[protocol.TrustZoneAuthKey]
			_, exists := b.GetKey(protocol.TrustZoneKey, n.Host().ID().String())
			for _, a := range pg.authProviders {
				a.Challenger(exists, c, n, b, trustAuth)
			}

			// 自动清理不在Hub中的对等节点的TZ
			if autocleanup {
				peers, err := n.MessageHub.ListPeers()
				if err != nil {
					return
				}
				tz := b.CurrentData()[protocol.TrustZoneKey]

				for k := range tz {
				PEER:
					for _, p := range peers {
						if p.String() == k {
							break PEER
						}
					}
					b.Delete(protocol.TrustZoneKey, k)
				}
			}
		})
		return nil
	}
}

// AuthProvider 是通用的区块链认证实体提供者接口
type AuthProvider interface {
	// Authenticate 要么生成稍后处理的挑战，要么根据区块链中可用的认证数据认证节点
	Authenticate(*hub.Message, chan *hub.Message, map[string]blockchain.Data) bool
	// Challenger 发送认证挑战
	Challenger(inTrustZone bool, c node.Config, n *node.Node, b *blockchain.Ledger, trustData map[string]blockchain.Data)
}
