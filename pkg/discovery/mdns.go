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

package discovery

import (
	"context"

	"github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
)

// MDNS mDNS发现服务结构体
type MDNS struct {
	DiscoveryServiceTag string // 发现服务标签
}

// discoveryNotifee 当通过mDNS发现发现新对等节点时收到通知
type discoveryNotifee struct {
	h host.Host          // libp2p主机
	c log.StandardLogger // 日志记录器
}

// HandlePeerFound 连接到通过mDNS发现的对等节点。一旦连接，
// PubSub系统将自动开始与它们交互（如果它们也支持PubSub）
func (n *discoveryNotifee) HandlePeerFound(pi peer.AddrInfo) {
	//n.c.Infof("mDNS: 发现新对等节点 %s\n", pi.ID.String())
	err := n.h.Connect(context.Background(), pi)
	if err != nil {
		n.c.Debugf("mDNS: 连接对等节点 %s 错误: %s\n", pi.ID.String(), err)
	}
}

// Option 返回libp2p选项
func (d *MDNS) Option(ctx context.Context) func(c *libp2p.Config) error {
	return func(*libp2p.Config) error { return nil }
}

// Run 运行mDNS发现服务
// 参数 l 为日志记录器，ctx 为上下文，host 为libp2p主机
func (d *MDNS) Run(l log.StandardLogger, ctx context.Context, host host.Host) error {
	// 设置mDNS发现以查找本地对等节点

	disc := mdns.NewMdnsService(host, d.DiscoveryServiceTag, &discoveryNotifee{h: host, c: l})
	return disc.Start()
}
