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
	"io"
	"net"
	"time"

	"github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/mudler/edgevpn/pkg/blockchain"
	"github.com/mudler/edgevpn/pkg/node"
	protocol "github.com/mudler/edgevpn/pkg/protocol"
	"github.com/pkg/errors"

	"github.com/mudler/edgevpn/pkg/types"
)

// ExposeNetworkService 暴露服务的网络服务
// 参数 announcetime 为公告时间间隔，serviceID 为服务ID
func ExposeNetworkService(announcetime time.Duration, serviceID string) node.NetworkService {
	return func(ctx context.Context, c node.Config, n *node.Node, b *blockchain.Ledger) error {
		b.Announce(
			ctx,
			announcetime,
			func() {
				// 从区块链中检索当前IP对应的ID
				existingValue, found := b.GetKey(protocol.ServicesLedgerKey, serviceID)
				service := &types.Service{}
				existingValue.Unmarshal(service)
				// 如果不匹配，则更新区块链
				if !found || service.PeerID != n.Host().ID().String() {
					updatedMap := map[string]interface{}{}
					updatedMap[serviceID] = types.Service{PeerID: n.Host().ID().String(), Name: serviceID}
					b.Add(protocol.ServicesLedgerKey, updatedMap)
				}
			},
		)
		return nil
	}
}

// RegisterService 将服务暴露到P2P网络。
// 应在节点使用Start()启动之前调用
// 参数 ll 为日志记录器，announcetime 为公告时间间隔，serviceID 为服务ID，dstaddress 为目标地址
func RegisterService(ll log.StandardLogger, announcetime time.Duration, serviceID, dstaddress string) []node.Option {
	ll.Infof("暴露服务 '%s' (%s)", serviceID, dstaddress)
	return []node.Option{
		node.WithStreamHandler(protocol.ServiceProtocol, func(n *node.Node, l *blockchain.Ledger) func(stream network.Stream) {
			return func(stream network.Stream) {
				go func() {
					ll.Infof("(服务 %s) 收到来自 %s 的连接", serviceID, stream.Conn().RemotePeer().String())

					// 从区块链中检索当前IP对应的ID
					_, found := l.GetKey(protocol.UsersLedgerKey, stream.Conn().RemotePeer().String())
					// 如果不匹配，则更新区块链
					if !found {
						ll.Debugf("重置 '%s': 在账本中未找到", stream.Conn().RemotePeer().String())
						stream.Reset()
						return
					}

					ll.Infof("正在连接到 '%s'", dstaddress)
					c, err := net.Dial("tcp", dstaddress)
					if err != nil {
						ll.Debugf("重置 %s: %s", stream.Conn().RemotePeer().String(), err.Error())
						stream.Reset()
						return
					}
					closer := make(chan struct{}, 2)
					go copyStream(closer, stream, c)
					go copyStream(closer, c, stream)
					<-closer

					stream.Close()
					c.Close()
					ll.Infof("(服务 %s) 正确处理 '%s'", serviceID, stream.Conn().RemotePeer().String())
				}()
			}
		}),
		node.WithNetworkService(ExposeNetworkService(announcetime, serviceID))}
}

// ConnectNetworkService 返回绑定到服务的网络服务
// 参数 announcetime 为公告时间间隔，serviceID 为服务ID，srcaddr 为源地址
func ConnectNetworkService(announcetime time.Duration, serviceID string, srcaddr string) node.NetworkService {
	return func(ctx context.Context, c node.Config, node *node.Node, ledger *blockchain.Ledger) error {
		// 打开本地端口进行监听
		l, err := net.Listen("tcp", srcaddr)
		if err != nil {
			return err
		}
		//	ll.Info("绑定本地端口到", srcaddr)

		// 公告我们自己，以便节点接受我们的连接
		ledger.Announce(
			ctx,
			announcetime,
			func() {
				// 从区块链中检索当前IP对应的ID
				_, found := ledger.GetKey(protocol.UsersLedgerKey, node.Host().ID().String())
				// 如果不匹配，则更新区块链
				if !found {
					updatedMap := map[string]interface{}{}
					updatedMap[node.Host().ID().String()] = &types.User{
						PeerID:    node.Host().ID().String(),
						Timestamp: time.Now().String(),
					}
					ledger.Add(protocol.UsersLedgerKey, updatedMap)
				}
			},
		)

		defer l.Close()
		for {
			select {
			case <-ctx.Done():
				return errors.New("上下文已取消")
			default:
				// 监听传入的连接
				conn, err := l.Accept()
				if err != nil {
					//	ll.Error("接受连接错误: ", err.Error())
					continue
				}

				//	ll.Info("新连接来自", l.Addr().String())
				// 在新的协程中处理连接，转发到P2P服务
				go func() {
					// 从区块链中检索当前IP对应的ID
					existingValue, found := ledger.GetKey(protocol.ServicesLedgerKey, serviceID)
					service := &types.Service{}
					existingValue.Unmarshal(service)
					// 如果不匹配，则更新区块链
					if !found {
						conn.Close()
						//	ll.Debugf("服务 '%s' 在区块链中未找到", serviceID)
						return
					}

					// 解码对等节点
					d, err := peer.Decode(service.PeerID)
					if err != nil {
						conn.Close()
						//	ll.Debugf("无法解码对等节点 '%s'", service.PeerID)
						return
					}

					// 打开流
					stream, err := node.Host().NewStream(ctx, d, protocol.ServiceProtocol.ID())
					if err != nil {
						conn.Close()
						//	ll.Debugf("无法打开流 '%s'", err.Error())
						return
					}
					//	ll.Debugf("(服务 %s) 正在重定向", serviceID, l.Addr().String())

					closer := make(chan struct{}, 2)
					go copyStream(closer, stream, conn)
					go copyStream(closer, conn, stream)
					<-closer

					stream.Close()
					conn.Close()
					//	ll.Infof("(服务 %s) 完成处理 %s", serviceID, l.Addr().String())
				}()
			}
		}

	}
}

// copyStream 复制流数据
// 参数 closer 为关闭通道，dst 为目标写入器，src 为源读取器
func copyStream(closer chan struct{}, dst io.Writer, src io.Reader) {
	defer func() { closer <- struct{}{} }() // 连接已关闭，发送信号停止代理
	io.Copy(dst, src)
}
