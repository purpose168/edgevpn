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
	"fmt"
	"sync"
	"time"

	"github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/p2p/net/conngater"

	"github.com/purpose168/edgevpn/pkg/crypto"
	protocol "github.com/purpose168/edgevpn/pkg/protocol"

	"github.com/purpose168/edgevpn/pkg/blockchain"
	hub "github.com/purpose168/edgevpn/pkg/hub"
	"github.com/purpose168/edgevpn/pkg/logger"
)

// Node 节点结构体，表示EdgeVPN网络中的一个节点
type Node struct {
	config     Config          // 节点配置
	MessageHub *hub.MessageHub // 消息中心

	//HubRoom *hub.Room
	inputCh      chan *hub.Message // 输入消息通道
	genericHubCh chan *hub.Message // 通用中心通道

	seed   int64                           // 随机种子
	host   host.Host                       // libp2p主机
	cg     *conngater.BasicConnectionGater // 连接门控器
	ledger *blockchain.Ledger              // 区块链账本
	sync.Mutex
}

// defaultChanSize 默认通道大小
const defaultChanSize = 3000

// defaultLibp2pOptions 默认libp2p选项
var defaultLibp2pOptions = []libp2p.Option{
	libp2p.EnableNATService(), // 启用NAT服务
	libp2p.NATPortMap(),       // 启用NAT端口映射
}

// New 创建新的节点实例
// 参数 p 为可选的配置选项
func New(p ...Option) (*Node, error) {
	c := &Config{
		DiscoveryInterval:        5 * time.Minute,                           // 发现间隔时间
		StreamHandlers:           make(map[protocol.Protocol]StreamHandler), // 流处理器映射
		LedgerAnnounceTime:       5 * time.Second,                           // 账本公告时间
		LedgerSyncronizationTime: 5 * time.Second,                           // 账本同步时间
		SealKeyLength:            defaultKeyLength,                          // 密钥长度
		Options:                  defaultLibp2pOptions,                      // libp2p选项
		Logger:                   logger.New(log.LevelDebug),                // 日志记录器
		Sealer:                   &crypto.AESSealer{},                       // 密封器
		Store:                    &blockchain.MemoryStore{},                 // 存储器
	}

	if err := c.Apply(p...); err != nil {
		return nil, err
	}

	return &Node{
		config:       *c,
		inputCh:      make(chan *hub.Message, defaultChanSize),
		genericHubCh: make(chan *hub.Message, defaultChanSize),
		seed:         0,
	}, nil
}

// Ledger 返回节点使用的账本
// 使用节点连接来广播消息
func (e *Node) Ledger() (*blockchain.Ledger, error) {
	e.Lock()
	defer e.Unlock()
	if e.ledger != nil {
		return e.ledger, nil
	}

	mw, err := e.messageWriter()
	if err != nil {
		return nil, err
	}

	e.ledger = blockchain.New(mw, e.config.Store)
	return e.ledger, nil
}

// PeerGater 返回节点的对等节点门控器
func (e *Node) PeerGater() Gater {
	return e.config.PeerGater
}

// Start 加入P2P网络启动节点
func (e *Node) Start(ctx context.Context) error {

	ledger, err := e.Ledger()
	if err != nil {
		return err
	}

	// 设置接收消息时的处理器
	// 账本需要读取它们并更新内部区块链
	e.config.Handlers = append(e.config.Handlers, ledger.Update)

	e.config.Logger.Info("正在启动 EdgeVPN 网络")

	// 启动 libp2p 网络
	err = e.startNetwork(ctx)
	if err != nil {
		return err
	}

	// 定期向通道发送包含我们区块链内容的消息
	ledger.Syncronizer(ctx, e.config.LedgerSyncronizationTime)

	// 启动声明的网络服务
	for _, s := range e.config.NetworkServices {
		err := s(ctx, e.config, e, ledger)
		if err != nil {
			return fmt.Errorf("启动网络服务时出错: '%w'", err)
		}
	}

	return nil
}

// messageWriter 返回绑定到edgevpn实例的新MessageWriter
// 使用给定的选项
func (e *Node) messageWriter(opts ...hub.MessageOption) (*messageWriter, error) {
	mess := &hub.Message{}
	mess.Apply(opts...)

	return &messageWriter{
		c:     e.config,
		input: e.inputCh,
		mess:  mess,
	}, nil
}

// startNetwork 启动网络
func (e *Node) startNetwork(ctx context.Context) error {
	e.config.Logger.Debug("生成主机数据")

	host, err := e.genHost(ctx)
	if err != nil {
		e.config.Logger.Error(err.Error())
		return err
	}
	e.host = host

	ledger, err := e.Ledger()
	if err != nil {
		return err
	}

	// 设置流处理器
	for pid, strh := range e.config.StreamHandlers {
		host.SetStreamHandler(pid.ID(), network.StreamHandler(strh(e, ledger)))
	}

	e.config.Logger.Info("节点 ID:", host.ID())
	e.config.Logger.Info("节点地址:", host.Addrs())

	// 中心在sealkey间隔内轮换
	// 这个时间长度应该足够进行几次区块交换。理想情况下是分钟级别（10、20等）
	// 它确保如果对加密消息尝试暴力破解，真实密钥不会被暴露
	e.MessageHub = hub.NewHub(e.config.RoomName, e.config.MaxMessageSize, e.config.SealKeyLength, e.config.SealKeyInterval, e.config.GenericHub)

	// 启动服务发现
	for _, sd := range e.config.ServiceDiscovery {
		if err := sd.Run(e.config.Logger, ctx, host); err != nil {
			e.config.Logger.Fatal(fmt.Errorf("启动服务发现时出错 %+v: '%w", sd, err))
		}
	}

	go e.handleEvents(ctx, e.inputCh, e.MessageHub.Messages, e.MessageHub.PublishMessage, e.config.Handlers, true)
	go e.MessageHub.Start(ctx, host)

	// 如果启用了通用中心，则单独创建一个，并关联一组通用通道处理器
	// 注意禁用对等节点门控，以便自由交换可用于认证或其他公共用途的消息
	if e.config.GenericHub {
		go e.handleEvents(ctx, e.genericHubCh, e.MessageHub.PublicMessages, e.MessageHub.PublishPublicMessage, e.config.GenericChannelHandler, false)
	}

	e.config.Logger.Debug("网络已启动")
	return nil
}

// PublishMessage 将消息发布到通用通道（如果已启用）
// 参见 GenericChannelHandlers(..) 来附加处理器以从此通道接收消息
func (e *Node) PublishMessage(m *hub.Message) error {
	if !e.config.GenericHub {
		return fmt.Errorf("通用中心已禁用")
	}

	e.genericHubCh <- m

	return nil
}
