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

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
	"github.com/mudler/edgevpn/internal"
	"github.com/mudler/edgevpn/pkg/config"
	"github.com/multiformats/go-multiaddr"

	"github.com/mudler/edgevpn/pkg/logger"
	node "github.com/mudler/edgevpn/pkg/node"
	"github.com/mudler/edgevpn/pkg/vpn"
	"github.com/urfave/cli/v2"
)

var CommonFlags []cli.Flag = []cli.Flag{
	&cli.StringFlag{
		Name:    "config",
		Usage:   "指定 edgevpn 配置文件路径",
		EnvVars: []string{"EDGEVPNCONFIG"},
	},
	&cli.StringSliceFlag{
		Name:    "listen-maddrs",
		Usage:   "覆盖默认的 0.0.0.0 监听多地址",
		EnvVars: []string{"EDGEVPNLISTENMADDRS"},
	},
	&cli.StringSliceFlag{
		Name:    "dht-announce-maddrs",
		Usage:   "在 DHT 公告时覆盖监听多地址",
		EnvVars: []string{"EDGEVPNDHTANNOUNCEMADDRS"},
	},
	&cli.StringFlag{
		Name:    "timeout",
		Usage:   "指定连接流的默认超时时间",
		EnvVars: []string{"EDGEVPNTIMEOUT"},
		Value:   "15s",
	},
	&cli.IntFlag{
		Name:    "mtu",
		Usage:   "指定 MTU（最大传输单元）",
		EnvVars: []string{"EDGEVPNMTU"},
		Value:   1200,
	},
	&cli.BoolFlag{
		Name:    "bootstrap-iface",
		Usage:   "启动时设置接口（需要权限）",
		EnvVars: []string{"EDGEVPNBOOTSTRAPIFACE"},
		Value:   true,
	},
	&cli.IntFlag{
		Name:    "packet-mtu",
		Usage:   "指定数据包 MTU",
		EnvVars: []string{"EDGEVPNPACKETMTU"},
		Value:   1420,
	},
	&cli.IntFlag{
		Name:    "channel-buffer-size",
		Usage:   "指定通道缓冲区大小",
		EnvVars: []string{"EDGEVPNCHANNELBUFFERSIZE"},
		Value:   0,
	},
	&cli.IntFlag{
		Name:    "discovery-interval",
		Usage:   "DHT 发现间隔时间",
		EnvVars: []string{"EDGEVPNDHTINTERVAL"},
		Value:   720,
	},
	&cli.IntFlag{
		Name:    "ledger-announce-interval",
		Usage:   "账本公告间隔时间",
		EnvVars: []string{"EDGEVPNLEDGERINTERVAL"},
		Value:   10,
	},
	&cli.StringFlag{
		Name:    "autorelay-discovery-interval",
		Usage:   "自动中继发现间隔",
		EnvVars: []string{"EDGEVPNAUTORELAYDISCOVERYINTERVAL"},
		Value:   "5m",
	},
	&cli.BoolFlag{
		Name:    "autorelay-static-only",
		Usage:   "仅使用定义的静态中继",
		EnvVars: []string{"EDGEVPNAUTORELAYSTATICONLY"},
	},
	&cli.IntFlag{
		Name:    "ledger-synchronization-interval",
		Usage:   "账本同步间隔时间",
		EnvVars: []string{"EDGEVPNLEDGERSYNCINTERVAL"},
		Value:   10,
	},
	&cli.IntFlag{
		Name:    "nat-ratelimit-global",
		Usage:   "全局请求速率限制",
		EnvVars: []string{"EDGEVPNNATRATELIMITGLOBAL"},
		Value:   10,
	},
	&cli.IntFlag{
		Name:    "nat-ratelimit-peer",
		Usage:   "对等节点请求速率限制",
		EnvVars: []string{"EDGEVPNNATRATELIMITPEER"},
		Value:   10,
	},
	&cli.IntFlag{
		Name:    "nat-ratelimit-interval",
		Usage:   "速率限制间隔",
		EnvVars: []string{"EDGEVPNNATRATELIMITINTERVAL"},
		Value:   60,
	},
	&cli.BoolFlag{
		Name:    "nat-ratelimit",
		Usage:   "更改帮助其他对等节点确定其可达性状态的默认速率限制配置",
		EnvVars: []string{"EDGEVPNNATRATELIMIT"},
		Value:   true,
	},
	&cli.IntFlag{
		Name:    "max-connections",
		Usage:   "最大连接数",
		EnvVars: []string{"EDGEVPNMAXCONNS"},
		Value:   0,
	},
	&cli.StringFlag{
		Name:    "ledger-state",
		Usage:   "指定账本状态目录",
		EnvVars: []string{"EDGEVPNLEDGERSTATE"},
	},
	&cli.BoolFlag{
		Name:    "mdns",
		Usage:   "启用 mDNS 进行对等节点发现",
		EnvVars: []string{"EDGEVPNMDNS"},
		Value:   true,
	},
	&cli.BoolFlag{
		Name:    "autorelay",
		Usage:   "如果节点可以接受入站连接，则自动充当中继",
		EnvVars: []string{"EDGEVPNAUTORELAY"},
		Value:   true,
	},
	&cli.IntFlag{
		Name:  "concurrency",
		Usage: "要服务的并发请求数",
		Value: runtime.NumCPU(),
	},
	&cli.BoolFlag{
		Name:    "holepunch",
		Usage:   "在可能时自动尝试打洞",
		EnvVars: []string{"EDGEVPNHOLEPUNCH"},
		Value:   true,
	},
	&cli.BoolFlag{
		Name:    "natservice",
		Usage:   "尝试确定节点的可达性状态",
		EnvVars: []string{"EDGEVPNNATSERVICE"},
		Value:   true,
	},
	&cli.BoolFlag{
		Name:    "natmap",
		Usage:   "尝试通过 UPnP 在防火墙中打开端口",
		EnvVars: []string{"EDGEVPNNATMAP"},
		Value:   true,
	},
	&cli.BoolFlag{
		Name:    "dht",
		Usage:   "启用 DHT 进行对等节点发现",
		EnvVars: []string{"EDGEVPNDHT"},
		Value:   true,
	},
	&cli.BoolFlag{
		Name:    "low-profile",
		Usage:   "启用低配置模式。降低连接使用量",
		EnvVars: []string{"EDGEVPNLOWPROFILE"},
		Value:   true,
	},
	&cli.IntFlag{
		Name:    "aliveness-healthcheck-interval",
		Usage:   "健康检查间隔",
		EnvVars: []string{"HEALTHCHECKINTERVAL"},
		Value:   120,
	},
	&cli.IntFlag{
		Name:    "aliveness-healthcheck-scrub-interval",
		Usage:   "健康检查清理间隔",
		EnvVars: []string{"HEALTHCHECKSCRUBINTERVAL"},
		Value:   600,
	},
	&cli.IntFlag{
		Name:    "aliveness-healthcheck-max-interval",
		Usage:   "健康检查最大间隔。判定节点离线的阈值",
		EnvVars: []string{"HEALTHCHECKMAXINTERVAL"},
		Value:   900,
	},
	&cli.StringFlag{
		Name:    "log-level",
		Usage:   "指定日志级别",
		EnvVars: []string{"EDGEVPNLOGLEVEL"},
		Value:   "info",
	},
	&cli.StringFlag{
		Name:    "libp2p-log-level",
		Usage:   "指定 libp2p 日志级别",
		EnvVars: []string{"EDGEVPNLIBP2PLOGLEVEL"},
		Value:   "fatal",
	},
	&cli.StringSliceFlag{
		Name:    "discovery-bootstrap-peers",
		Usage:   "要使用的发现对等节点列表",
		EnvVars: []string{"EDGEVPNBOOTSTRAPPEERS"},
	},
	&cli.IntFlag{
		Name:    "connection-high-water",
		Usage:   "允许的最大连接数",
		EnvVars: []string{"EDGEVPN_CONNECTION_HIGH_WATER"},
		Value:   0,
	},
	&cli.IntFlag{
		Name:    "connection-low-water",
		Usage:   "允许的最小连接数",
		EnvVars: []string{"EDGEVPN_CONNECTION_LOW_WATER"},
		Value:   0,
	},
	&cli.StringSliceFlag{
		Name:    "autorelay-static-peer",
		Usage:   "要使用的自动中继静态对等节点列表",
		EnvVars: []string{"EDGEVPNAUTORELAYPEERS"},
	},
	&cli.StringSliceFlag{
		Name:    "blacklist",
		Usage:   "要限制的对等节点/CIDR 列表",
		EnvVars: []string{"EDGEVPNBLACKLIST"},
	},
	&cli.StringFlag{
		Name:    "token",
		Usage:   "指定 edgevpn 令牌以代替配置文件",
		EnvVars: []string{"EDGEVPNTOKEN"},
	},
	&cli.BoolFlag{
		Name:    "limit-enable",
		Usage:   "启用资源管理",
		EnvVars: []string{"LIMITENABLE"},
	},
	&cli.StringFlag{
		Name:    "limit-file",
		Usage:   "指定资源限制配置文件（JSON）",
		EnvVars: []string{"LIMITFILE"},
	},
	&cli.StringFlag{
		Name:    "limit-scope",
		Usage:   "指定限制范围",
		EnvVars: []string{"LIMITSCOPE"},
		Value:   "system",
	},
	&cli.IntFlag{
		Name:    "limit-config-streams",
		Usage:   "流资源限制配置",
		EnvVars: []string{"LIMITCONFIGSTREAMS"},
		Value:   200,
	},
	&cli.IntFlag{
		Name:    "limit-config-streams-inbound",
		Usage:   "入站流资源限制配置",
		EnvVars: []string{"LIMITCONFIGSTREAMSINBOUND"},
		Value:   30,
	},
	&cli.IntFlag{
		Name:    "limit-config-streams-outbound",
		Usage:   "出站流资源限制配置",
		EnvVars: []string{"LIMITCONFIGSTREAMSOUTBOUND"},
		Value:   30,
	},
	&cli.IntFlag{
		Name:    "limit-config-conn",
		Usage:   "连接资源限制配置",
		EnvVars: []string{"LIMITCONFIGCONNS"},
		Value:   200,
	},
	&cli.IntFlag{
		Name:    "limit-config-conn-inbound",
		Usage:   "入站连接资源限制配置",
		EnvVars: []string{"LIMITCONFIGCONNSINBOUND"},
		Value:   30,
	},
	&cli.IntFlag{
		Name:    "limit-config-conn-outbound",
		Usage:   "出站连接资源限制配置",
		EnvVars: []string{"LIMITCONFIGCONNSOUTBOUND"},
		Value:   30,
	},
	&cli.IntFlag{
		Name:    "limit-config-fd",
		Usage:   "最大文件描述符资源限制配置",
		EnvVars: []string{"LIMITCONFIGFD"},
		Value:   30,
	},
	&cli.BoolFlag{
		Name:    "peerguard",
		Usage:   "启用对等节点保护。（实验性）",
		EnvVars: []string{"PEERGUARD"},
	},
	&cli.BoolFlag{
		Name:    "privkey-cache",
		Usage:   "启用私钥缓存。（实验性）",
		EnvVars: []string{"EDGEVPNPRIVKEYCACHE"},
	},
	&cli.StringFlag{
		Name:    "privkey-cache-dir",
		Usage:   "指定用于存储生成的私钥的目录",
		EnvVars: []string{"EDGEVPNPRIVKEYCACHEDIR"},
		Value:   stateDir(),
	},
	&cli.StringSliceFlag{
		Name:    "static-peertable",
		Usage:   "要使用的静态对等节点列表（格式为 `ip:peerid`）",
		EnvVars: []string{"EDGEVPNSTATICPEERTABLE"},
	},
	&cli.StringSliceFlag{
		Name:    "whitelist",
		Usage:   "白名单中的对等节点列表",
		EnvVars: []string{"EDGEVPNWHITELIST"},
	},
	&cli.BoolFlag{
		Name:    "peergate",
		Usage:   "启用对等节点门控。（实验性）",
		EnvVars: []string{"PEERGATE"},
	},
	&cli.BoolFlag{
		Name:    "peergate-autoclean",
		Usage:   "启用对等节点门控自动清理。（实验性）",
		EnvVars: []string{"PEERGATE_AUTOCLEAN"},
	},
	&cli.BoolFlag{
		Name:    "peergate-relaxed",
		Usage:   "启用对等节点门控宽松模式。（实验性）",
		EnvVars: []string{"PEERGATE_RELAXED"},
	},
	&cli.StringFlag{
		Name:    "peergate-auth",
		Usage:   "对等节点门控认证",
		EnvVars: []string{"PEERGATE_AUTH"},
		Value:   "",
	},
	&cli.IntFlag{
		Name:    "peergate-interval",
		Usage:   "对等节点门控间隔时间",
		EnvVars: []string{"EDGEVPNPEERGATEINTERVAL"},
		Value:   120,
	},
}

func stateDir() string {
	baseDir := ".edgevpn"
	home, _ := os.UserHomeDir()
	if home != "" {
		return filepath.Join(home, baseDir)
	}

	dir, _ := os.Getwd()
	return filepath.Join(dir, baseDir)
}

func displayStart(ll *logger.Logger) {
	ll.Info(Copyright)

	ll.Infof("版本: %s 提交: %s", internal.Version, internal.Commit)
}

func stringsToMultiAddr(peers []string) []multiaddr.Multiaddr {
	res := []multiaddr.Multiaddr{}
	for _, p := range peers {
		addr, err := multiaddr.NewMultiaddr(p)
		if err != nil {
			continue
		}
		res = append(res, addr)
	}
	return res
}

// ConfigFromContext 从 CLI 上下文返回配置对象
func ConfigFromContext(c *cli.Context) *config.Config {
	var limitConfig *rcmgr.PartialLimitConfig

	autorelayInterval, err := time.ParseDuration(c.String("autorelay-discovery-interval"))
	if err != nil {
		autorelayInterval = 0
	}

	// 认证提供者应该作为 JSON 对象传递
	pa := c.String("peergate-auth")
	d := map[string]map[string]interface{}{}
	json.Unmarshal([]byte(pa), &d)

	return &config.Config{
		NetworkConfig:     c.String("config"),
		NetworkToken:      c.String("token"),
		ListenMaddrs:      (c.StringSlice("listen-maddrs")),
		DHTAnnounceMaddrs: stringsToMultiAddr(c.StringSlice("dht-announce-maddrs")),
		Address:           c.String("address"),
		Router:            c.String("router"),
		Interface:         c.String("interface"),
		Libp2pLogLevel:    c.String("libp2p-log-level"),
		LogLevel:          c.String("log-level"),
		LowProfile:        c.Bool("low-profile"),
		Blacklist:         c.StringSlice("blacklist"),
		Concurrency:       c.Int("concurrency"),
		FrameTimeout:      c.String("timeout"),
		ChannelBufferSize: c.Int("channel-buffer-size"),
		InterfaceMTU:      c.Int("mtu"),
		PacketMTU:         c.Int("packet-mtu"),
		BootstrapIface:    c.Bool("bootstrap-iface"),
		Whitelist:         stringsToMultiAddr(c.StringSlice("whitelist")),
		Ledger: config.Ledger{
			StateDir:         c.String("ledger-state"),
			AnnounceInterval: time.Duration(c.Int("ledger-announce-interval")) * time.Second,
			SyncInterval:     time.Duration(c.Int("ledger-synchronization-interval")) * time.Second,
		},
		NAT: config.NAT{
			Service:           c.Bool("natservice"),
			Map:               c.Bool("natmap"),
			RateLimit:         c.Bool("nat-ratelimit"),
			RateLimitGlobal:   c.Int("nat-ratelimit-global"),
			RateLimitPeer:     c.Int("nat-ratelimit-peer"),
			RateLimitInterval: time.Duration(c.Int("nat-ratelimit-interval")) * time.Second,
		},
		Discovery: config.Discovery{
			BootstrapPeers: c.StringSlice("discovery-bootstrap-peers"),
			DHT:            c.Bool("dht"),
			MDNS:           c.Bool("mdns"),
			Interval:       time.Duration(c.Int("discovery-interval")) * time.Second,
		},
		Connection: config.Connection{
			AutoRelay:                  c.Bool("autorelay"),
			MaxConnections:             c.Int("max-connections"),
			HolePunch:                  c.Bool("holepunch"),
			StaticRelays:               c.StringSlice("autorelay-static-peer"),
			AutoRelayDiscoveryInterval: autorelayInterval,
			OnlyStaticRelays:           c.Bool("autorelay-static-only"),
			HighWater:                  c.Int("connection-high-water"),
			LowWater:                   c.Int("connection-low-water"),
		},
		Limit: config.ResourceLimit{
			Enable:      c.Bool("limit-enable"),
			FileLimit:   c.String("limit-file"),
			Scope:       c.String("limit-scope"),
			MaxConns:    c.Int("max-connections"), // 设置为 0 以使用其他限制方式。文件优先
			LimitConfig: limitConfig,
		},
		PeerGuard: config.PeerGuard{
			Enable:        c.Bool("peerguard"),
			PeerGate:      c.Bool("peergate"),
			Relaxed:       c.Bool("peergate-relaxed"),
			Autocleanup:   c.Bool("peergate-autoclean"),
			SyncInterval:  time.Duration(c.Int("peergate-interval")) * time.Second,
			AuthProviders: d,
		},
	}
}

func cliToOpts(c *cli.Context) ([]node.Option, []vpn.Option, *logger.Logger) {
	nc := ConfigFromContext(c)

	lvl, err := log.LevelFromString(nc.LogLevel)
	if err != nil {
		lvl = log.LevelError
	}
	llger := logger.New(lvl)

	checkErr := func(e error) {
		if err != nil {
			llger.Fatal(err.Error())
		}
	}

	// 检查我们是否已经缓存了任何私钥身份
	if c.Bool("privkey-cache") {
		keyFile := filepath.Join(c.String("privkey-cache-dir"), "privkey")
		dat, err := os.ReadFile(keyFile)
		if err == nil && len(dat) > 0 {
			llger.Info("从", keyFile, "读取密钥")

			nc.Privkey = dat
		} else {
			// 生成并写入
			llger.Info("生成私钥并保存到", keyFile, "以供后续使用")

			privkey, err := node.GenPrivKey(0)
			checkErr(err)

			r, err := crypto.MarshalPrivateKey(privkey)
			checkErr(err)

			err = os.MkdirAll(c.String("privkey-cache-dir"), 0600)
			checkErr(err)

			err = os.WriteFile(keyFile, r, 0600)
			checkErr(err)

			nc.Privkey = r
		}
	}

	for _, pt := range c.StringSlice("static-peertable") {
		dat := strings.Split(pt, ":")
		if len(dat) != 2 {
			checkErr(fmt.Errorf("对等节点表条目格式错误。需要以 `:` 分隔的 ip/peerid 列表。例如 10.1.0.1:... "))
		}
		if nc.Connection.PeerTable == nil {
			nc.Connection.PeerTable = make(map[string]peer.ID)
		}

		nc.Connection.PeerTable[dat[0]] = peer.ID(dat[1])
	}

	nodeOpts, vpnOpts, err := nc.ToOpts(llger)
	if err != nil {
		llger.Fatal(err.Error())
	}

	return nodeOpts, vpnOpts, llger
}

func handleStopSignals() {
	s := make(chan os.Signal, 10)
	signal.Notify(s, os.Interrupt, syscall.SIGTERM)

	for range s {
		os.Exit(0)
	}
}
