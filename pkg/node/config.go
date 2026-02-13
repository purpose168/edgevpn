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
	"time"

	"github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/mudler/edgevpn/pkg/blockchain"
	discovery "github.com/mudler/edgevpn/pkg/discovery"
	hub "github.com/mudler/edgevpn/pkg/hub"
	protocol "github.com/mudler/edgevpn/pkg/protocol"
)

// Config 节点配置结构体
type Config struct {
	// ExchangeKey 是用于密封消息的对称密钥
	ExchangeKey string

	// RoomName 是所有对等节点订阅的OTP令牌八卦房间
	RoomName string

	// ListenAddresses 是发现对等节点的初始引导地址
	ListenAddresses []discovery.AddrList

	// Insecure 禁用安全的P2P端到端加密通信
	Insecure bool

	// Handlers 是订阅VPN接口接收消息的处理器列表
	Handlers, GenericChannelHandler []Handler

	MaxMessageSize  int // 最大消息大小
	SealKeyInterval int // 密封密钥间隔

	ServiceDiscovery []ServiceDiscovery // 服务发现列表
	NetworkServices  []NetworkService   // 网络服务列表
	Logger           log.StandardLogger // 日志记录器

	SealKeyLength    int    // 密封密钥长度
	InterfaceAddress string // 接口地址

	Store blockchain.Store // 存储器

	// Handle 是被HumanInterfaces消耗的句柄，用于处理接收的消息
	Handle                     func(bool, *hub.Message)
	StreamHandlers             map[protocol.Protocol]StreamHandler // 流处理器映射
	AdditionalOptions, Options []libp2p.Option                     // libp2p选项

	DiscoveryInterval, LedgerSyncronizationTime, LedgerAnnounceTime time.Duration // 各种时间间隔
	DiscoveryBootstrapPeers                                         discovery.AddrList // 发现引导节点

	Whitelist, Blacklist []string // 白名单和黑名单

	// GenericHub 启用通用中心
	GenericHub bool

	PrivateKey []byte            // 私钥
	PeerTable  map[string]peer.ID // 对等节点表

	Sealer    Sealer // 密封器
	PeerGater Gater  // 对等节点门控器
}

// Gater 对等节点门控器接口
type Gater interface {
	Gate(*Node, peer.ID) bool // 门控检查
	Enable()                   // 启用
	Disable()                  // 禁用
	Enabled() bool             // 是否启用
}

// Sealer 密封器接口
type Sealer interface {
	Seal(string, string) (string, error)   // 密封
	Unseal(string, string) (string, error) // 解封
}

// NetworkService 是运行在网络上的服务。它接收上下文、节点和账本
type NetworkService func(context.Context, Config, *Node, *blockchain.Ledger) error

// StreamHandler 流处理器类型
type StreamHandler func(*Node, *blockchain.Ledger) func(stream network.Stream)

// Handler 消息处理器类型
type Handler func(*blockchain.Ledger, *hub.Message, chan *hub.Message) error

// ServiceDiscovery 服务发现接口
type ServiceDiscovery interface {
	Run(log.StandardLogger, context.Context, host.Host) error        // 运行服务发现
	Option(context.Context) func(c *libp2p.Config) error             // 返回libp2p选项
}

// Option 配置选项函数类型
type Option func(cfg *Config) error

// Apply 应用给定的选项到配置，返回遇到的第一个错误（如果有）
func (cfg *Config) Apply(opts ...Option) error {
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(cfg); err != nil {
			return err
		}
	}
	return nil
}
