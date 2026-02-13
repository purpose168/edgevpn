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
	"crypto/sha256"
	"sync"
	"time"

	internalCrypto "github.com/mudler/edgevpn/pkg/crypto"
	"github.com/mudler/edgevpn/pkg/utils"

	"github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/routing"
	discovery "github.com/libp2p/go-libp2p/p2p/discovery/routing"
)

// DHT DHT发现服务结构体
type DHT struct {
	OTPKey               string        // OTP密钥
	OTPInterval          int           // OTP间隔
	KeyLength            int           // 密钥长度
	RendezvousString     string        // 会合点字符串
	BootstrapPeers       AddrList      // 引导节点列表
	rendezvousHistory    Ring          // 会合点历史
	RefreshDiscoveryTime time.Duration // 刷新发现时间
	*dht.IpfsDHT                       // DHT实例
	dhtOptions           []dht.Option  // DHT选项
}

// NewDHT 创建新的DHT发现服务
// 参数 d 为DHT选项
func NewDHT(d ...dht.Option) *DHT {
	return &DHT{dhtOptions: d, rendezvousHistory: Ring{Length: 2}}
}

// Option 返回libp2p选项
func (d *DHT) Option(ctx context.Context) func(c *libp2p.Config) error {
	return libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
		// 使用给定的主机创建DHT
		return d.startDHT(ctx, h)
	})
}

// Rendezvous 返回会合点字符串
func (d *DHT) Rendezvous() string {
	if d.OTPKey != "" {
		totp := internalCrypto.TOTP(sha256.New, d.KeyLength, d.OTPInterval, d.OTPKey)
		rv := internalCrypto.MD5(totp)
		return rv
	}
	return d.RendezvousString
}

// startDHT 启动DHT
// 参数 ctx 为上下文，h 为libp2p主机
func (d *DHT) startDHT(ctx context.Context, h host.Host) (*dht.IpfsDHT, error) {
	if d.IpfsDHT == nil {
		// 启动DHT用于对等节点发现。我们不能只创建新的DHT客户端
		// 因为我们希望每个对等节点维护自己的DHT本地副本，
		// 这样DHT的引导节点可以下线而不影响未来的对等节点发现

		kad, err := dht.New(ctx, h, d.dhtOptions...)
		if err != nil {
			return d.IpfsDHT, err
		}
		d.IpfsDHT = kad
	}

	return d.IpfsDHT, nil
}

// announceRendezvous 公告会合点
// 参数 c 为日志记录器，ctx 为上下文，host 为libp2p主机，kademliaDHT 为DHT实例
func (d *DHT) announceRendezvous(c log.StandardLogger, ctx context.Context, host host.Host, kademliaDHT *dht.IpfsDHT) {
	d.bootstrapPeers(c, ctx, host)
	rv := d.Rendezvous()
	d.rendezvousHistory.Add(rv)

	c.Debugf("正在使用以下会合点: %+v", d.rendezvousHistory.Data)
	for _, r := range d.rendezvousHistory.Data {
		c.Debugf("使用会合点公告: %s", r)
		d.announceAndConnect(c, ctx, kademliaDHT, host, r)
	}
	c.Debug("会合点公告完成")
}

// Run 运行DHT发现服务
// 参数 c 为日志记录器，ctx 为上下文，host 为libp2p主机
func (d *DHT) Run(c log.StandardLogger, ctx context.Context, host host.Host) error {
	if d.KeyLength == 0 {
		d.KeyLength = 12
	}

	if len(d.BootstrapPeers) == 0 {
		d.BootstrapPeers = dht.DefaultBootstrapPeers
	}
	// 启动DHT用于对等节点发现。我们不能只创建新的DHT客户端
	// 因为我们希望每个对等节点维护自己的DHT本地副本，
	// 这样DHT的引导节点可以下线而不影响未来的对等节点发现
	kademliaDHT, err := d.startDHT(ctx, host)
	if err != nil {
		return err
	}

	// 引导DHT。在默认配置中，这会生成一个后台线程
	// 每五分钟刷新一次对等节点表
	c.Info("正在引导DHT")
	if err = kademliaDHT.Bootstrap(ctx); err != nil {
		return err
	}

	go d.runBackground(c, ctx, host, kademliaDHT)

	return nil
}

// runBackground 运行后台任务
// 参数 c 为日志记录器，ctx 为上下文，host 为libp2p主机，kademliaDHT 为DHT实例
func (d *DHT) runBackground(c log.StandardLogger, ctx context.Context, host host.Host, kademliaDHT *dht.IpfsDHT) {
	d.announceRendezvous(c, ctx, host, kademliaDHT)
	t := utils.NewBackoffTicker(utils.BackoffMaxInterval(d.RefreshDiscoveryTime))
	defer t.Stop()
	for {
		select {
		case <-t.C:
			// 我们向所有对等节点的会合点公告自己
			// 我们有1小时的安全保障，避免在网络问题时阻塞主循环
			// DHT的TTL默认不超过3小时，所以我们的条目应该安全
			safeTimeout, cancel := context.WithTimeout(ctx, time.Hour)

			endChan := make(chan struct{})
			go func() {
				d.announceRendezvous(c, safeTimeout, host, kademliaDHT)
				endChan <- struct{}{}
			}()

			select {
			case <-endChan:
				cancel()
			case <-safeTimeout.Done():
				c.Error("公告会合点超时")
				cancel()
			}
		case <-ctx.Done():
			return
		}
	}
}

// bootstrapPeers 连接引导节点
// 参数 c 为日志记录器，ctx 为上下文，host 为libp2p主机
func (d *DHT) bootstrapPeers(c log.StandardLogger, ctx context.Context, host host.Host) {
	// 首先连接到引导节点。它们会告诉我们网络中的其他节点
	var wg sync.WaitGroup
	for _, peerAddr := range d.BootstrapPeers {
		peerinfo, _ := peer.AddrInfoFromP2pAddr(peerAddr)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if host.Network().Connectedness(peerinfo.ID) != network.Connected {
				if err := host.Connect(ctx, *peerinfo); err != nil {
					c.Debug(err.Error())
				} else {
					c.Debug("与引导节点建立连接:", *peerinfo)
				}
			}
		}()
	}
	wg.Wait()
}

// FindClosePeers 查找附近的对等节点
// 参数 ll 为日志记录器，onlyStaticRelays 为是否仅使用静态中继，static 为静态中继列表
func (d *DHT) FindClosePeers(ll log.StandardLogger, onlyStaticRelays bool, static ...string) func(ctx context.Context, numPeers int) <-chan peer.AddrInfo {
	return func(ctx context.Context, numPeers int) <-chan peer.AddrInfo {
		peerChan := make(chan peer.AddrInfo, numPeers)
		go func() {

			toStream := []peer.AddrInfo{}

			if !onlyStaticRelays {
				closestPeers, err := d.GetClosestPeers(ctx, d.PeerID().String())
				if err != nil {
					ll.Debug("获取最近对等节点错误: ", err)
				}

				for _, p := range closestPeers {
					addrs := d.Host().Peerstore().Addrs(p)
					if len(addrs) == 0 {
						continue
					}
					ll.Debugf("[中继发现] 找到附近对等节点 '%s'", p.String())
					toStream = append(toStream, peer.AddrInfo{ID: p, Addrs: addrs})
				}
			}

			for _, r := range static {
				pi, err := peer.AddrInfoFromString(r)
				if err == nil {
					ll.Debug("[静态中继发现] 扫描 ", pi.ID)
					toStream = append(toStream, peer.AddrInfo{ID: pi.ID, Addrs: pi.Addrs})
				}
			}

			if len(toStream) > numPeers {
				toStream = toStream[0 : numPeers-1]
			}

			for _, t := range toStream {
				peerChan <- t
			}

			close(peerChan)
		}()

		return peerChan
	}
}

// announceAndConnect 公告并连接到对等节点
// 参数 l 为日志记录器，ctx 为上下文，kademliaDHT 为DHT实例，host 为libp2p主机，rv 为会合点
func (d *DHT) announceAndConnect(l log.StandardLogger, ctx context.Context, kademliaDHT *dht.IpfsDHT, host host.Host, rv string) error {
	l.Debug("正在公告自己...")

	tCtx, c := context.WithTimeout(ctx, time.Second*120)
	defer c()
	routingDiscovery := discovery.NewRoutingDiscovery(kademliaDHT)
	routingDiscovery.Advertise(tCtx, rv)
	l.Debug("成功公告!")
	// 现在，查找其他已公告的对等节点
	// 这就像你的朋友告诉你见面的地点
	l.Debug("正在搜索其他对等节点...")

	fCtx, cf := context.WithTimeout(ctx, time.Second*120)
	defer cf()
	peerChan, err := routingDiscovery.FindPeers(fCtx, rv)
	if err != nil {
		return err
	}

	for p := range peerChan {
		// 不要拨号自己或没有地址的对等节点
		if p.ID == host.ID() || len(p.Addrs) == 0 {
			continue
		}

		if host.Network().Connectedness(p.ID) != network.Connected {
			l.Debug("找到对等节点:", p)
			timeoutCtx, cancel := context.WithTimeout(ctx, time.Second*120)
			defer cancel()
			if err := host.Connect(timeoutCtx, p); err != nil {
				l.Debugf("连接 '%s' 失败，错误: '%s'", p, err.Error())
			} else {
				l.Debug("已连接到:", p)
			}
		} else {
			l.Debug("已知对等节点（已连接）:", p)
		}
	}

	l.Debug("完成对等节点搜索")

	return nil
}
