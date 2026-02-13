// Copyright © 2021-2022 Ettore Di Giacinto <mudler@mocaccino.org>
//
// This program is free software; you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation; either version 2 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License along
// with this program; if not, see <http://www.gnu.org/licenses/>.

package services

import (
	"context"
	"time"

	"github.com/mudler/edgevpn/pkg/node"
	"github.com/mudler/edgevpn/pkg/protocol"
	"github.com/mudler/edgevpn/pkg/utils"

	"github.com/mudler/edgevpn/pkg/blockchain"
)

// AliveNetworkService 存活检测网络服务
// 参数 announcetime 为公告时间间隔，scrubTime 为清理时间间隔，maxtime 为最大超时时间
func AliveNetworkService(announcetime, scrubTime, maxtime time.Duration) node.NetworkService {
	return func(ctx context.Context, c node.Config, n *node.Node, b *blockchain.Ledger) error {
		t := time.Now()
		// 通过定期向区块链公告我们的服务
		b.Announce(
			ctx,
			announcetime,
			func() {
				// 保持活跃
				b.Add(protocol.HealthCheckKey, map[string]interface{}{
					n.Host().ID().String(): time.Now().UTC().Format(time.RFC3339),
				})

				// 保持活跃清理
				nodes := AvailableNodes(b, maxtime)
				if len(nodes) == 0 {
					return
				}
				lead := utils.Leader(nodes)
				if !t.Add(scrubTime).After(time.Now()) {
					// 更新计时器，防止非领导者在之后尝试删除存储桶
					// 防止循环
					t = time.Now()

					if lead == n.Host().ID().String() {
						// 一段时间后自动清理
						b.DeleteBucket(protocol.HealthCheckKey)
					}
				}
			},
		)
		return nil
	}
}

// Alive 每隔公告时间公告节点，并定期清理健康检查
// maxtime 用于确定节点何时不可达（超过maxtime后，节点被视为不可达）
// 参数 announcetime 为公告时间间隔，scrubTime 为清理时间间隔，maxtime 为最大超时时间
func Alive(announcetime, scrubTime, maxtime time.Duration) []node.Option {
	return []node.Option{
		node.WithNetworkService(AliveNetworkService(announcetime, scrubTime, maxtime)),
	}
}

// AvailableNodes 返回在最近maxTime时间内发送过健康检查的可用节点
// 参数 b 为区块链账本，maxTime 为最大时间窗口
func AvailableNodes(b *blockchain.Ledger, maxTime time.Duration) (active []string) {
	for u, t := range b.LastBlock().Storage[protocol.HealthCheckKey] {
		var s string
		t.Unmarshal(&s)
		parsed, _ := time.Parse(time.RFC3339, s)
		if parsed.Add(maxTime).After(time.Now().UTC()) {
			active = append(active, u)
		}
	}

	return active
}
