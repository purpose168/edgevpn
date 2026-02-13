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

package vpn

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/purpose168/edgevpn/internal"
	"github.com/purpose168/edgevpn/pkg/blockchain"
	"github.com/purpose168/edgevpn/pkg/logger"
	"github.com/purpose168/edgevpn/pkg/node"
	"github.com/purpose168/edgevpn/pkg/protocol"
	"github.com/purpose168/edgevpn/pkg/stream"
	"github.com/purpose168/edgevpn/pkg/types"

	"github.com/mudler/water"
	"github.com/pkg/errors"
	"github.com/songgao/packets/ethernet"
)

// streamManager 流管理器接口，定义了流连接、断开、检查和关闭的方法
type streamManager interface {
	Connected(n network.Network, c network.Stream)
	Disconnected(n network.Network, c network.Stream)
	HasStream(n network.Network, pid peer.ID) (network.Stream, error)
	Close() error
}

// VPNNetworkService 返回一个VPN网络服务，用于处理VPN连接和数据转发
// 参数 p 为可选的配置选项
func VPNNetworkService(p ...Option) node.NetworkService {
	return func(ctx context.Context, nc node.Config, n *node.Node, b *blockchain.Ledger) error {
		// 初始化默认配置
		c := &Config{
			Concurrency:        1,                          // 并发数
			LedgerAnnounceTime: 5 * time.Second,            // 账本公告时间
			Timeout:            15 * time.Second,           // 超时时间
			Logger:             logger.New(log.LevelDebug), // 日志记录器
			MaxStreams:         30,                         // 最大流数量
		}
		// 应用配置选项
		if err := c.Apply(p...); err != nil {
			return err
		}

		// 创建网络接口
		ifce, err := createInterface(c)
		if err != nil {
			return err
		}
		defer ifce.Close()

		var mgr streamManager

		if c.lowProfile {
			// 为出站连接创建流管理器
			mgr, err = stream.NewConnManager(10, c.MaxStreams)
			if err != nil {
				return err
			}
			// 将其附加到相同的上下文
			go func() {
				<-ctx.Done()
				mgr.Close()
			}()
		}

		// 在运行时设置流处理器
		n.Host().SetStreamHandler(protocol.EdgeVPN.ID(), streamHandler(b, ifce, c, nc))

		// 公告我们的IP地址
		ip, _, err := net.ParseCIDR(c.InterfaceAddress)
		if err != nil {
			return err
		}

		// 定期向账本公告我们的IP信息
		b.Announce(
			ctx,
			c.LedgerAnnounceTime,
			func() {
				machine := &types.Machine{}
				// 从区块链中检索当前IP对应的ID
				existingValue, found := b.GetKey(protocol.MachinesLedgerKey, ip.String())
				existingValue.Unmarshal(machine)

				// 如果不匹配，则更新区块链
				if !found || machine.PeerID != n.Host().ID().String() {
					updatedMap := map[string]interface{}{}
					updatedMap[ip.String()] = newBlockChainData(n, ip.String())
					b.Add(protocol.MachinesLedgerKey, updatedMap)
				}
			},
		)

		// 如果启用了NetLink引导，则准备网络接口
		if c.NetLinkBootstrap {
			if err := prepareInterface(c); err != nil {
				return err
			}
		}

		// 从接口读取数据包
		return readPackets(ctx, mgr, c, n, b, ifce, nc)
	}
}

// Start the node and the vpn. Returns an error in case of failure
// When starting the vpn, there is no need to start the node
// Register 注册VPN服务，返回节点选项
// 启动节点和VPN。如果失败则返回错误
// 启动VPN时，无需单独启动节点
func Register(p ...Option) ([]node.Option, error) {
	return []node.Option{node.WithNetworkService(VPNNetworkService(p...))}, nil
}

// streamHandler 返回一个流处理函数，用于处理传入的数据流
// 参数 l 为区块链账本，ifce 为网络接口，c 为配置，nc 为节点配置
func streamHandler(l *blockchain.Ledger, ifce *water.Interface, c *Config, nc node.Config) func(stream network.Stream) {
	return func(stream network.Stream) {
		// 检查对等节点是否在允许列表中
		if len(nc.PeerTable) == 0 && !l.Exists(protocol.MachinesLedgerKey,
			func(d blockchain.Data) bool {
				machine := &types.Machine{}
				d.Unmarshal(machine)
				return machine.PeerID == stream.Conn().RemotePeer().String()
			}) {
			stream.Reset()
			return
		}
		// 如果有对等节点表，则检查是否在表中
		if len(nc.PeerTable) > 0 {
			found := false
			for _, p := range nc.PeerTable {
				if p.String() == stream.Conn().RemotePeer().String() {
					found = true
				}
			}
			if !found {
				stream.Reset()
				return
			}
		}
		// 将流数据复制到网络接口
		_, err := io.Copy(ifce.ReadWriteCloser, stream)
		if err != nil {
			stream.Reset()
		}
		stream.Close()
	}
}

// newBlockChainData 创建新的区块链数据，包含节点信息
// 参数 n 为节点实例，address 为IP地址
func newBlockChainData(n *node.Node, address string) types.Machine {
	hostname, _ := os.Hostname()

	return types.Machine{
		PeerID:   n.Host().ID().String(), // 对等节点ID
		Hostname: hostname,               // 主机名
		OS:       runtime.GOOS,           // 操作系统
		Arch:     runtime.GOARCH,         // 架构
		Version:  internal.Version,       // 版本
		Address:  address,                // IP地址
	}
}

// getFrame 从网络接口读取以太网帧
// 参数 ifce 为网络接口，c 为配置
func getFrame(ifce *water.Interface, c *Config) (ethernet.Frame, error) {
	var frame ethernet.Frame
	frame.Resize(c.MTU)

	n, err := ifce.Read([]byte(frame))
	if err != nil {
		return frame, errors.Wrap(err, "无法从接口读取数据")
	}

	frame = frame[:n]
	return frame, nil
}

// handleFrame 处理以太网帧，将其转发到目标对等节点
// 参数 mgr 为流管理器，frame 为以太网帧，c 为配置，n 为节点，ip 为本地IP，ledger 为账本，ifce 为接口，nc 为节点配置
func handleFrame(mgr streamManager, frame ethernet.Frame, c *Config, n *node.Node, ip net.IP, ledger *blockchain.Ledger, ifce *water.Interface, nc node.Config) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.Timeout)
	defer cancel()

	var dstIP, srcIP net.IP
	var packet layers.IPv4
	// 尝试解析IPv4数据包
	if err := packet.DecodeFromBytes(frame, gopacket.NilDecodeFeedback); err != nil {
		var packet layers.IPv6
		// 尝试解析IPv6数据包
		if err := packet.DecodeFromBytes(frame, gopacket.NilDecodeFeedback); err != nil {
			return errors.Wrap(err, "无法从帧中解析头部信息")
		} else {
			dstIP = packet.DstIP
			srcIP = packet.SrcIP
		}
	} else {
		dstIP = packet.DstIP
		srcIP = packet.SrcIP
	}

	dst := dstIP.String()
	// 如果配置了路由地址且源IP是本地IP，则检查目标是否在账本中
	if c.RouterAddress != "" && srcIP.Equal(ip) {
		if _, found := ledger.GetKey(protocol.MachinesLedgerKey, dst); !found {
			dst = c.RouterAddress
		}
	}

	var d peer.ID
	var err error
	notFoundErr := fmt.Errorf("路由表中未找到 '%s'", dst)
	// 检查对等节点表
	if len(nc.PeerTable) > 0 {
		found := false
		for ip, p := range nc.PeerTable {
			if ip == dst {
				found = true
				d = peer.ID(p)
			}
		}
		if !found {
			return notFoundErr
		}
	} else {
		// 查询路由表
		value, found := ledger.GetKey(protocol.MachinesLedgerKey, dst)
		if !found {
			return notFoundErr
		}
		machine := &types.Machine{}
		value.Unmarshal(machine)

		// 解码对等节点ID
		d, err = peer.Decode(machine.PeerID)
	}

	if err != nil {
		return errors.Wrap(err, "无法解码对等节点")
	}

	var stream network.Stream
	if mgr != nil {
		// 如果需要，打开一个流
		stream, err = mgr.HasStream(n.Host().Network(), d)
		if err == nil {
			_, err = stream.Write(frame)
			if err == nil {
				return nil
			}
			mgr.Disconnected(n.Host().Network(), stream)
		}
	}

	// 创建新的流连接
	stream, err = n.Host().NewStream(ctx, d, protocol.EdgeVPN.ID())
	if err != nil {
		return fmt.Errorf("无法打开到 %s 的流: %w", d.String(), err)
	}
	defer stream.Close()

	if mgr != nil {
		mgr.Connected(n.Host().Network(), stream)
	}

	_, err = stream.Write(frame)
	return err
}

// connectionWorker 连接工作协程，从通道中读取帧并处理
// 参数 p 为帧通道，mgr 为流管理器，c 为配置，n 为节点，ip 为本地IP，wg 为等待组，ledger 为账本，ifce 为接口，nc 为节点配置
func connectionWorker(
	p chan ethernet.Frame,
	mgr streamManager,
	c *Config,
	n *node.Node,
	ip net.IP,
	wg *sync.WaitGroup,
	ledger *blockchain.Ledger,
	ifce *water.Interface,
	nc node.Config) {
	defer wg.Done()
	for f := range p {
		if err := handleFrame(mgr, f, c, n, ip, ledger, ifce, nc); err != nil {
			c.Logger.Debugf("无法处理帧: %s", err.Error())
		}
	}
}

// readPackets 从接口读取数据包，并使用区块链中的路由表将其转发到节点
// 参数 ctx 为上下文，mgr 为流管理器，c 为配置，n 为节点，ledger 为账本，ifce 为接口，nc 为节点配置
func readPackets(ctx context.Context, mgr streamManager, c *Config, n *node.Node, ledger *blockchain.Ledger, ifce *water.Interface, nc node.Config) error {
	ip, _, err := net.ParseCIDR(c.InterfaceAddress)
	if err != nil {
		return err
	}

	wg := new(sync.WaitGroup)

	packets := make(chan ethernet.Frame, c.ChannelBufferSize)

	defer func() {
		close(packets)
		wg.Wait()
	}()

	// 启动多个并发工作协程处理数据包
	for i := 0; i < c.Concurrency; i++ {
		wg.Add(1)
		go connectionWorker(packets, mgr, c, n, ip, wg, ledger, ifce, nc)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			frame, err := getFrame(ifce, c)
			if err != nil {
				c.Logger.Errorf("无法获取帧 '%s'", err.Error())
				continue
			}

			packets <- frame
		}
	}
}
