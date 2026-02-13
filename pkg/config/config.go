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

package config

import (
	"fmt"
	"math/bits"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/host/autorelay"
	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
	connmanager "github.com/libp2p/go-libp2p/p2p/net/connmgr"
	"github.com/mudler/water"
	"github.com/multiformats/go-multiaddr"
	"github.com/peterbourgon/diskv"
	"github.com/purpose168/edgevpn/pkg/blockchain"
	"github.com/purpose168/edgevpn/pkg/crypto"
	"github.com/purpose168/edgevpn/pkg/discovery"
	"github.com/purpose168/edgevpn/pkg/logger"
	"github.com/purpose168/edgevpn/pkg/node"
	"github.com/purpose168/edgevpn/pkg/trustzone"
	"github.com/purpose168/edgevpn/pkg/trustzone/authprovider/ecdsa"
	"github.com/purpose168/edgevpn/pkg/vpn"
)

// Config 是节点和默认EdgeVPN服务的配置结构体
// 用于在启动前为节点和服务生成选项
type Config struct {
	NetworkConfig, NetworkToken                string                // 网络配置和网络令牌
	Address                                    string                // IP地址
	ListenMaddrs                               []string              // 监听多地址
	DHTAnnounceMaddrs                          []multiaddr.Multiaddr // DHT公告多地址
	Router                                     string                // 路由器地址
	Interface                                  string                // 接口名称
	Libp2pLogLevel, LogLevel                   string                // libp2p日志级别和日志级别
	LowProfile, BootstrapIface                 bool                  // 低配置模式和引导接口
	Blacklist                                  []string              // 黑名单
	Concurrency                                int                   // 并发数
	FrameTimeout                               string                // 帧超时
	ChannelBufferSize, InterfaceMTU, PacketMTU int                   // 通道缓冲区大小、接口MTU、数据包MTU
	NAT                                        NAT                   // NAT配置
	Connection                                 Connection            // 连接配置
	Discovery                                  Discovery             // 发现配置
	Ledger                                     Ledger                // 账本配置
	Limit                                      ResourceLimit         // 资源限制
	Privkey                                    []byte                // 私钥
	// PeerGuard (实验性)
	// 启用peerguardian并添加特定的认证选项
	PeerGuard PeerGuard

	Whitelist []multiaddr.Multiaddr // 白名单
}

// PeerGuard 对等节点保护配置
type PeerGuard struct {
	Enable      bool // 是否启用
	Relaxed     bool // 宽松模式
	Autocleanup bool // 自动清理
	PeerGate    bool // 对等节点门控
	// AuthProviders 以自由映射形式提供认证提供者：
	// ecdsa:
	//   private_key: "foo_bar"
	AuthProviders map[string]map[string]interface{} // 认证提供者配置
	SyncInterval  time.Duration                     // 同步间隔
}

// ResourceLimit 资源限制配置
type ResourceLimit struct {
	FileLimit   string                    // 限制文件路径
	LimitConfig *rcmgr.PartialLimitConfig // 限制配置
	Scope       string                    // 作用域
	MaxConns    int                       // 最大连接数
	StaticMin   int64                     // 静态最小值
	StaticMax   int64                     // 静态最大值
	Enable      bool                      // 是否启用
}

// Ledger 是账本配置结构
type Ledger struct {
	AnnounceInterval, SyncInterval time.Duration // 公告间隔和同步间隔
	StateDir                       string        // 状态目录
}

// Discovery 允许启用/禁用发现并设置引导节点
type Discovery struct {
	DHT, MDNS      bool          // 是否启用DHT和mDNS发现
	BootstrapPeers []string      // 引导节点列表
	Interval       time.Duration // 发现间隔
}

// Connection 是连接服务相关的配置部分
type Connection struct {
	HolePunch bool // 是否启用打洞
	AutoRelay bool // 是否启用自动中继

	AutoRelayDiscoveryInterval time.Duration // 自动中继发现间隔
	StaticRelays               []string      // 静态中继列表
	OnlyStaticRelays           bool          // 仅使用静态中继

	PeerTable map[string]peer.ID // 对等节点表

	MaxConnections int // 最大连接数

	LowWater  int // 低水位线
	HighWater int // 高水位线
}

// NAT 是NAT配置设置相关的结构
// 允许启用/禁用服务和NAT映射，以及速率限制
type NAT struct {
	Service   bool // 是否启用NAT服务
	Map       bool // 是否启用NAT映射
	RateLimit bool // 是否启用速率限制

	RateLimitGlobal, RateLimitPeer int           // 全局速率限制和对等节点速率限制
	RateLimitInterval              time.Duration // 速率限制间隔
}

// Validate 验证配置是否有效，无效则返回错误
func (c Config) Validate() error {
	if c.NetworkConfig == "" &&
		c.NetworkToken == "" {
		return fmt.Errorf("未提供EDGEVPNCONFIG或EDGEVPNTOKEN。至少需要一个配置文件")
	}
	return nil
}

// peers2List 将对等节点字符串列表转换为发现地址列表
func peers2List(peers []string) discovery.AddrList {
	addrsList := discovery.AddrList{}
	for _, p := range peers {
		addrsList.Set(p)
	}
	return addrsList
}

// peers2AddrInfo 将对等节点字符串列表转换为对等节点地址信息列表
func peers2AddrInfo(peers []string) []peer.AddrInfo {
	addrsList := []peer.AddrInfo{}
	for _, p := range peers {
		pi, err := peer.AddrInfoFromString(p)
		if err == nil {
			addrsList = append(addrsList, *pi)
		}

	}
	return addrsList
}

var infiniteResourceLimits = rcmgr.InfiniteLimits.ToPartialLimitConfig().System

// ToOpts 从配置返回节点和VPN选项
// 参数 l 为日志记录器
func (c Config) ToOpts(l *logger.Logger) ([]node.Option, []vpn.Option, error) {

	if err := c.Validate(); err != nil {
		return nil, nil, err
	}

	config := c.NetworkConfig
	address := c.Address
	router := c.Router
	iface := c.Interface
	logLevel := c.LogLevel
	libp2plogLevel := c.Libp2pLogLevel
	dhtE, mDNS := c.Discovery.DHT, c.Discovery.MDNS

	ledgerState := c.Ledger.StateDir

	peers := c.Discovery.BootstrapPeers

	// 解析日志级别
	lvl, err := log.LevelFromString(logLevel)
	if err != nil {
		lvl = log.LevelError
	}

	llger := logger.New(lvl)

	// 解析libp2p日志级别
	libp2plvl, err := log.LevelFromString(libp2plogLevel)
	if err != nil {
		libp2plvl = log.LevelFatal
	}

	token := c.NetworkToken

	addrsList := peers2List(peers)

	dhtOpts := []dht.Option{}

	if c.LowProfile {
		dhtOpts = append(dhtOpts, dht.BucketSize(20))
	}
	if len(c.DHTAnnounceMaddrs) > 0 {
		dhtOpts = append(dhtOpts, dht.AddressFilter(
			func(m []multiaddr.Multiaddr) []multiaddr.Multiaddr {
				return c.DHTAnnounceMaddrs
			},
		),
		)
	}

	d := discovery.NewDHT(dhtOpts...)
	m := &discovery.MDNS{}

	opts := []node.Option{
		node.ListenAddresses(c.ListenMaddrs...),
		node.WithDiscoveryInterval(c.Discovery.Interval),
		node.WithLedgerAnnounceTime(c.Ledger.AnnounceInterval),
		node.WithLedgerInterval(c.Ledger.SyncInterval),
		node.Logger(llger),
		node.WithDiscoveryBootstrapPeers(addrsList),
		node.WithBlacklist(c.Blacklist...),
		node.LibP2PLogLevel(libp2plvl),
		node.WithInterfaceAddress(address),
		node.WithSealer(&crypto.AESSealer{}),
		node.FromBase64(mDNS, dhtE, token, d, m),
		node.FromYaml(mDNS, dhtE, config, d, m),
	}

	// 添加静态对等节点
	for ip, peer := range c.Connection.PeerTable {
		opts = append(opts, node.WithStaticPeer(ip, peer))
	}

	// 添加私钥
	if len(c.Privkey) > 0 {
		opts = append(opts, node.WithPrivKey(c.Privkey))
	}

	vpnOpts := []vpn.Option{
		vpn.WithConcurrency(c.Concurrency),
		vpn.WithInterfaceAddress(address),
		vpn.WithLedgerAnnounceTime(c.Ledger.AnnounceInterval),
		vpn.Logger(llger),
		vpn.WithTimeout(c.FrameTimeout),
		vpn.WithInterfaceType(water.TUN),
		vpn.NetLinkBootstrap(c.BootstrapIface),
		vpn.WithChannelBufferSize(c.ChannelBufferSize),
		vpn.WithInterfaceMTU(c.InterfaceMTU),
		vpn.WithPacketMTU(c.PacketMTU),
		vpn.WithRouterAddress(router),
		vpn.WithInterfaceName(iface),
	}

	libp2pOpts := []libp2p.Option{libp2p.UserAgent("edgevpn")}

	// AutoRelay部分配置
	if c.Connection.AutoRelay {
		relayOpts := []autorelay.Option{}

		staticRelays := c.Connection.StaticRelays

		if c.Connection.AutoRelayDiscoveryInterval == 0 {
			c.Connection.AutoRelayDiscoveryInterval = 5 * time.Minute
		}
		// 如果没有指定中继且没有发现间隔，则使用默认静态中继（将被弃用）

		relayOpts = append(relayOpts, autorelay.WithPeerSource(d.FindClosePeers(llger, c.Connection.OnlyStaticRelays, staticRelays...)))

		libp2pOpts = append(libp2pOpts,
			libp2p.EnableAutoRelay(relayOpts...))
	}

	// NAT速率限制配置
	if c.NAT.RateLimit {
		libp2pOpts = append(libp2pOpts, libp2p.AutoNATServiceRateLimit(
			c.NAT.RateLimitGlobal,
			c.NAT.RateLimitPeer,
			c.NAT.RateLimitInterval,
		))
	}

	// 连接管理器配置
	if c.Connection.LowWater != 0 && c.Connection.HighWater != 0 {
		llger.Infof("连接管理器水位线限制 低: %d 高: %d", c.Connection.LowWater, c.Connection.HighWater)

		cm, err := connmanager.NewConnManager(
			c.Connection.LowWater,
			c.Connection.HighWater,
			connmanager.WithGracePeriod(80*time.Second),
		)
		if err != nil {
			llger.Fatal("无法创建连接管理器")
		}

		libp2pOpts = append(libp2pOpts, libp2p.ConnectionManager(cm))
	} else {
		llger.Infof("连接管理器已禁用")
	}

	// 资源管理器配置
	if !c.Limit.Enable || runtime.GOOS == "darwin" {
		llger.Info("go-libp2p资源管理器保护已禁用")
		libp2pOpts = append(libp2pOpts, libp2p.ResourceManager(&network.NullResourceManager{}))
	} else {
		llger.Info("go-libp2p资源管理器保护已启用")

		var limiter rcmgr.Limiter

		if c.Limit.FileLimit != "" {
			// 从JSON文件加载限制配置
			limitFile, err := os.Open(c.Limit.FileLimit)
			if err != nil {
				return opts, vpnOpts, err
			}
			defer limitFile.Close()

			l, err := rcmgr.NewDefaultLimiterFromJSON(limitFile)
			if err != nil {
				return opts, vpnOpts, err
			}

			limiter = l
		} else if c.Limit.MaxConns == -1 {
			llger.Infof("最大连接数: 无限制")

			scalingLimits := rcmgr.DefaultLimits

			// 为包含的libp2p协议添加限制
			libp2p.SetDefaultServiceLimits(&scalingLimits)

			// 使用`.AutoScale`将缩放限制转换为具体的限制集
			// 这会根据系统内存按比例缩放限制
			scaledDefaultLimits := scalingLimits.AutoScale()

			// 调整某些设置
			cfg := rcmgr.PartialLimitConfig{
				System: rcmgr.ResourceLimits{
					Memory: rcmgr.Unlimited64,
					FD:     rcmgr.Unlimited,

					Conns:         rcmgr.Unlimited,
					ConnsInbound:  rcmgr.Unlimited,
					ConnsOutbound: rcmgr.Unlimited,

					Streams:         rcmgr.Unlimited,
					StreamsOutbound: rcmgr.Unlimited,
					StreamsInbound:  rcmgr.Unlimited,
				},

				// 瞬态连接不会导致资源管理器/会计记录任何内存
				// 只有已建立的连接才会
				// 因此，我们不能依赖System.Memory来保护我们免受大量瞬态连接的影响
				// 我们限制与System作用域相同的值，但只允许Transient作用域占用System作用域允许的25%
				Transient: rcmgr.ResourceLimits{
					Memory:        rcmgr.Unlimited64,
					FD:            rcmgr.Unlimited,
					Conns:         rcmgr.Unlimited,
					ConnsInbound:  rcmgr.Unlimited,
					ConnsOutbound: rcmgr.Unlimited,

					Streams:         rcmgr.Unlimited,
					StreamsInbound:  rcmgr.Unlimited,
					StreamsOutbound: rcmgr.Unlimited,
				},

				// 让我们不要妨碍白名单功能
				// 如果有人指定了"Swarm.ResourceMgr.Allowlist"，我们应该让它通过
				AllowlistedSystem: infiniteResourceLimits,

				AllowlistedTransient: infiniteResourceLimits,

				// 保持简单，不设置Service、ServicePeer、Protocol、ProtocolPeer、Conn或Stream限制
				ServiceDefault: infiniteResourceLimits,

				ServicePeerDefault: infiniteResourceLimits,

				ProtocolDefault: infiniteResourceLimits,

				ProtocolPeerDefault: infiniteResourceLimits,

				Conn: infiniteResourceLimits,

				Stream: infiniteResourceLimits,

				// 限制单个对等节点消耗的资源
				// 这不能保护我们免受故意的DoS攻击，因为攻击者可以轻松启动多个对等节点
				// 我们指定此限制是为了防止无意的DoS攻击（例如，对等节点有bug并故意发送过多流量）
				// 在这种情况下，我们希望控制该对等节点的资源消耗
				// 为保持简单，我们只限制入站连接和流
				PeerDefault: rcmgr.ResourceLimits{
					Memory:          rcmgr.Unlimited64,
					FD:              rcmgr.Unlimited,
					Conns:           rcmgr.Unlimited,
					ConnsInbound:    rcmgr.DefaultLimit,
					ConnsOutbound:   rcmgr.Unlimited,
					Streams:         rcmgr.Unlimited,
					StreamsInbound:  rcmgr.DefaultLimit,
					StreamsOutbound: rcmgr.Unlimited,
				},
			}

			// 使用我们的cfg创建限制，并用`scaledDefaultLimits`中的值替换默认值
			limits := cfg.Build(scaledDefaultLimits)

			// 资源管理器需要一个限制器，所以我们从限制创建一个
			limiter = rcmgr.NewFixedLimiter(limits)

		} else if c.Limit.MaxConns != 0 {
			min := int64(1 << 30)
			max := int64(4 << 30)
			if c.Limit.StaticMin != 0 {
				min = c.Limit.StaticMin
			}
			if c.Limit.StaticMax != 0 {
				max = c.Limit.StaticMax
			}
			maxconns := int(c.Limit.MaxConns)

			defaultLimits := rcmgr.DefaultLimits.Scale(min+max/2, logScale(2*maxconns))
			llger.Infof("最大连接数: %d", c.Limit.MaxConns)

			limiter = rcmgr.NewFixedLimiter(defaultLimits)
		} else {
			llger.Infof("最大连接数: 默认限制")

			defaults := rcmgr.DefaultLimits
			def := &defaults

			libp2p.SetDefaultServiceLimits(def)
			limiter = rcmgr.NewFixedLimiter(def.AutoScale())
		}

		rc, err := rcmgr.NewResourceManager(limiter, rcmgr.WithAllowlistedMultiaddrs(c.Whitelist))
		if err != nil {
			llger.Fatal("无法创建资源管理器")
		}

		libp2pOpts = append(libp2pOpts, libp2p.ResourceManager(rc))
	}

	// 打洞配置
	if c.Connection.HolePunch {
		libp2pOpts = append(libp2pOpts, libp2p.EnableHolePunching())
	}

	// NAT服务配置
	if c.NAT.Service {
		libp2pOpts = append(libp2pOpts, libp2p.EnableNATService())
	}

	// NAT端口映射配置
	if c.NAT.Map {
		libp2pOpts = append(libp2pOpts, libp2p.NATPortMap())
	}

	opts = append(opts, node.WithLibp2pOptions(libp2pOpts...))

	// 账本存储配置
	if ledgerState != "" {
		opts = append(opts, node.WithStore(blockchain.NewDiskStore(diskv.New(diskv.Options{
			BasePath:     ledgerState,
			CacheSizeMax: uint64(50), // 50MB
		}))))
	} else {
		opts = append(opts, node.WithStore(&blockchain.MemoryStore{}))
	}

	// 对等节点保护配置
	if c.PeerGuard.Enable {
		pg := trustzone.NewPeerGater(c.PeerGuard.Relaxed)
		dur := c.PeerGuard.SyncInterval

		// 为peerguardian构建认证提供者
		aps := []trustzone.AuthProvider{}
		for ap, providerOpts := range c.PeerGuard.AuthProviders {
			a, err := authProvider(llger, ap, providerOpts)
			if err != nil {
				return opts, vpnOpts, fmt.Errorf("无效的认证提供者: %w", err)
			}
			aps = append(aps, a)
		}

		pguardian := trustzone.NewPeerGuardian(llger, aps...)

		opts = append(opts,
			node.WithNetworkService(
				pg.UpdaterService(dur),
				pguardian.Challenger(dur, c.PeerGuard.Autocleanup),
			),
			node.EnableGenericHub,
			node.GenericChannelHandlers(pguardian.ReceiveMessage),
		)
		// 我们总是传递一个PeerGater，以便在必要时注册到API
		opts = append(opts, node.WithPeerGater(pg))
		// 如果未启用，我们立即禁用它
		if !c.PeerGuard.PeerGate {
			pg.Disable()
		}
	}

	return opts, vpnOpts, nil
}

// authProvider 创建认证提供者
// 参数 ll 为日志记录器，s 为提供者类型，opts 为选项
func authProvider(ll log.StandardLogger, s string, opts map[string]interface{}) (trustzone.AuthProvider, error) {
	switch strings.ToLower(s) {
	case "ecdsa":
		pk, exists := opts["private_key"]
		if !exists {
			return nil, fmt.Errorf("未提供私钥")
		}
		return ecdsa.ECDSA521Provider(ll, fmt.Sprint(pk))
	}
	return nil, fmt.Errorf("不支持")
}

// logScale 计算对数缩放值
func logScale(val int) int {
	bitlen := bits.Len(uint(val))
	return 1 << bitlen
}
