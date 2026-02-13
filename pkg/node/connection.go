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

package node

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"io"
	mrand "math/rand"
	"net"

	internalCrypto "github.com/purpose168/edgevpn/pkg/crypto"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	conngater "github.com/libp2p/go-libp2p/p2p/net/conngater"
	"github.com/libp2p/go-libp2p/p2p/net/swarm"
	multiaddr "github.com/multiformats/go-multiaddr"
	hub "github.com/purpose168/edgevpn/pkg/hub"
)

// Host 返回libp2p对等节点主机
func (e *Node) Host() host.Host {
	return e.host
}

// ConnectionGater 返回底层libp2p连接门控器
func (e *Node) ConnectionGater() *conngater.BasicConnectionGater {
	return e.cg
}

// BlockSubnet 阻止CIDR子网的连接
// 参数 cidr 为CIDR格式的子网地址
func (e *Node) BlockSubnet(cidr string) error {
	// 避免通过VPN尝试连接节点导致回环流量
	_, n, err := net.ParseCIDR(cidr)
	if err != nil {
		return err
	}

	return e.ConnectionGater().BlockSubnet(n)
}

// GenPrivKey 生成私钥
// 参数 seed 为随机种子，为0时使用加密随机数
func GenPrivKey(seed int64) (crypto.PrivKey, error) {
	var r io.Reader
	if seed == 0 {
		r = rand.Reader
	} else {
		r = mrand.New(mrand.NewSource(seed))
	}
	prvKey, _, err := crypto.GenerateKeyPairWithReader(crypto.Ed25519, 4096, r)
	return prvKey, err
}

// genHost 生成libp2p主机
func (e *Node) genHost(ctx context.Context) (host.Host, error) {
	var prvKey crypto.PrivKey

	opts := e.config.Options

	cg, err := conngater.NewBasicConnectionGater(nil)
	if err != nil {
		return nil, err
	}

	e.cg = cg

	// 阻止VPN接口地址的连接
	if e.config.InterfaceAddress != "" {
		e.BlockSubnet(e.config.InterfaceAddress)
	}

	// 处理黑名单
	for _, b := range e.config.Blacklist {
		_, net, err := net.ParseCIDR(b)
		if err != nil {
			// 假设是对等节点ID
			cg.BlockPeer(peer.ID(b))
		}
		if net != nil {
			cg.BlockSubnet(net)
		}
	}

	// 如果未指定则生成私钥
	if len(e.config.PrivateKey) > 0 {
		prvKey, err = crypto.UnmarshalPrivateKey(e.config.PrivateKey)
	} else {
		prvKey, err = GenPrivKey(e.seed)
	}

	if err != nil {
		return nil, err
	}

	opts = append(opts, libp2p.ConnectionGater(cg), libp2p.Identity(prvKey))
	// 暂时不启用指标
	opts = append(opts, libp2p.DisableMetrics())

	// 设置监听地址
	addrs := []multiaddr.Multiaddr{}
	for _, l := range e.config.ListenAddresses {
		addrs = append(addrs, []multiaddr.Multiaddr(l)...)
	}
	opts = append(opts, libp2p.ListenAddrs(addrs...))

	// 添加服务发现选项
	for _, d := range e.config.ServiceDiscovery {
		opts = append(opts, d.Option(ctx))
	}

	opts = append(opts, e.config.AdditionalOptions...)

	// 如果启用不安全模式，禁用安全传输层
	if e.config.Insecure {
		e.config.Logger.Info("正在禁用安全传输层")
		opts = append(opts, libp2p.NoSecurity)
	}

	opts = append(opts, FallbackDefaults)

	return libp2p.NewWithoutDefaults(opts...)
}

// FallbackDefaults 如果没有应用其他相关选项，则将默认选项应用于libp2p节点
// 将附加到传递给New的选项中
var FallbackDefaults libp2p.Option = func(cfg *libp2p.Config) error {
	for _, def := range defaults {
		if !def.fallback(cfg) {
			continue
		}
		if err := cfg.Apply(def.opt); err != nil {
			return err
		}
	}
	return nil
}

// defaultUDPBlackHoleDetector 默认UDP黑洞检测器
var defaultUDPBlackHoleDetector = func(cfg *libp2p.Config) error {
	// 黑洞是一个二元属性。在网络上如果UDP拨号被阻止，所有拨号都会失败
	// 所以100次拨号中5次成功的低成功率就足够了
	return cfg.Apply(libp2p.UDPBlackHoleSuccessCounter(&swarm.BlackHoleSuccessCounter{N: 100, MinSuccesses: 5, Name: "UDP"}))
}

// defaultIPv6BlackHoleDetector 默认IPv6黑洞检测器
var defaultIPv6BlackHoleDetector = func(cfg *libp2p.Config) error {
	// 黑洞是一个二元属性。在网络上如果没有IPv6连接，所有拨号都会失败
	// 所以100次拨号中5次成功的低成功率就足够了
	return cfg.Apply(libp2p.IPv6BlackHoleSuccessCounter(&swarm.BlackHoleSuccessCounter{N: 100, MinSuccesses: 5, Name: "IPv6"}))
}

// defaults 默认选项及其回退条件的完整列表
//
// 请*不要*以其他任何方式指定默认选项。将所有内容放在这里
// 使跟踪默认值*更加容易*
// https://github.com/libp2p/go-libp2p/blob/2209ae05976df6a1cc2631c961f57549d109008c/defaults.go#L227
var defaults = []struct {
	fallback func(cfg *libp2p.Config) bool
	opt      libp2p.Option
}{
	{
		fallback: func(cfg *libp2p.Config) bool { return cfg.Transports == nil && cfg.ListenAddrs == nil },
		opt:      libp2p.DefaultListenAddrs,
	},
	{
		fallback: func(cfg *libp2p.Config) bool { return cfg.Transports == nil && cfg.PSK == nil },
		opt:      libp2p.DefaultTransports,
	},
	{
		fallback: func(cfg *libp2p.Config) bool { return cfg.Transports == nil && cfg.PSK != nil },
		opt:      libp2p.DefaultPrivateTransports,
	},
	{
		fallback: func(cfg *libp2p.Config) bool { return cfg.Muxers == nil },
		opt:      libp2p.DefaultMuxers,
	},
	{
		fallback: func(cfg *libp2p.Config) bool { return !cfg.Insecure && cfg.SecurityTransports == nil },
		opt:      libp2p.DefaultSecurity,
	},
	{
		fallback: func(cfg *libp2p.Config) bool { return cfg.PeerKey == nil },
		opt:      libp2p.RandomIdentity,
	},
	{
		fallback: func(cfg *libp2p.Config) bool { return cfg.Peerstore == nil },
		opt:      libp2p.DefaultPeerstore,
	},
	{
		fallback: func(cfg *libp2p.Config) bool { return !cfg.RelayCustom },
		opt:      libp2p.DefaultEnableRelay,
	},
	{
		fallback: func(cfg *libp2p.Config) bool { return cfg.ResourceManager == nil },
		opt:      libp2p.DefaultResourceManager,
	},
	//{
	//	fallback: func(cfg *libp2p.Config) bool { return cfg.ResourceManager == nil },
	//	opt:      libp2p.DefaultResourceManager,
	//},
	{
		fallback: func(cfg *libp2p.Config) bool { return cfg.ConnManager == nil },
		// 填充ConnManager是必需的，即使是空的管理器，因为libp2p会调用
		// libp2p.Config.ConnManager的函数，所以我们需要它不为nil
		opt: libp2p.DefaultConnectionManager,
		//opt: libp2p.ConnectionManager(connmgr.NullConnMgr{}),
	},
	{
		fallback: func(cfg *libp2p.Config) bool {
			return !cfg.CustomUDPBlackHoleSuccessCounter && cfg.UDPBlackHoleSuccessCounter == nil
		},
		opt: defaultUDPBlackHoleDetector,
	},
	{
		fallback: func(cfg *libp2p.Config) bool {
			return !cfg.CustomIPv6BlackHoleSuccessCounter && cfg.IPv6BlackHoleSuccessCounter == nil
		},
		opt: defaultIPv6BlackHoleDetector,
	},
	//{
	//	fallback: func(cfg *libp2p.Config) bool { return !cfg.DisableMetrics && cfg.PrometheusRegisterer == nil },
	//	opt:      libp2p.DefaultPrometheusRegisterer,
	//},
}

// sealkey 生成密封密钥
func (e *Node) sealkey() string {
	return internalCrypto.MD5(internalCrypto.TOTP(sha256.New, e.config.SealKeyLength, e.config.SealKeyInterval, e.config.ExchangeKey))
}

// handleEvents 处理事件循环
// 参数 ctx 为上下文，inputChannel 为输入通道，roomMessages 为房间消息通道，pub 为发布函数，handlers 为处理器列表，peerGater 为是否启用对等节点门控
func (e *Node) handleEvents(ctx context.Context, inputChannel chan *hub.Message, roomMessages chan *hub.Message, pub func(*hub.Message) error, handlers []Handler, peerGater bool) {
	for {
		select {
		case m := <-inputChannel:
			if m == nil {
				continue
			}
			c := m.Copy()
			str, err := e.config.Sealer.Seal(c.Message, e.sealkey())
			if err != nil {
				e.config.Logger.Warnf("%w 来自 %s", err.Error(), c.SenderID)
			}
			c.Message = str

			if err := pub(c); err != nil {
				e.config.Logger.Warnf("发布错误: %s", err)
			}

		case m := <-roomMessages:
			if m == nil {
				continue
			}

			// 对等节点门控检查
			if peerGater {
				if e.config.PeerGater != nil && e.config.PeerGater.Gate(e, peer.ID(m.SenderID)) {
					e.config.Logger.Warnf("已门控来自 %s 的消息", m.SenderID)
					continue
				}
			}
			// 对等节点表检查
			if len(e.config.PeerTable) > 0 {
				found := false
				for _, p := range e.config.PeerTable {
					if p.String() == peer.ID(m.SenderID).String() {
						found = true
					}
				}
				if !found {
					e.config.Logger.Warnf("已门控来自 %s 的消息 - 不在对等节点表中", m.SenderID)
					continue
				}
			}

			c := m.Copy()
			str, err := e.config.Sealer.Unseal(c.Message, e.sealkey())
			if err != nil {
				e.config.Logger.Warnf("%w 来自 %s", err.Error(), c.SenderID)
			}
			c.Message = str
			e.handleReceivedMessage(c, handlers, inputChannel)
		case <-ctx.Done():
			return
		}
	}
}

// handleReceivedMessage 处理接收到的消息
// 参数 m 为消息，handlers 为处理器列表，c 为消息通道
func (e *Node) handleReceivedMessage(m *hub.Message, handlers []Handler, c chan *hub.Message) {
	for _, h := range handlers {
		if err := h(e.ledger, m, c); err != nil {
			e.config.Logger.Warnf("处理器错误: %s", err)
		}
	}
}
