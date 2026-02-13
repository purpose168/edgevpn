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
	"os"
	"time"

	"github.com/ipfs/go-log"
	"github.com/purpose168/edgevpn/pkg/node"
	"github.com/purpose168/edgevpn/pkg/protocol"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"
	"github.com/purpose168/edgevpn/pkg/blockchain"
	"github.com/purpose168/edgevpn/pkg/types"
)

// SharefileNetworkService 共享文件网络服务
// 参数 announcetime 为公告时间间隔，fileID 为文件ID
func SharefileNetworkService(announcetime time.Duration, fileID string) node.NetworkService {
	return func(ctx context.Context, c node.Config, n *node.Node, b *blockchain.Ledger) error {
		// 通过定期向区块链公告我们的服务
		b.Announce(
			ctx,
			announcetime,
			func() {
				// 从区块链中检索当前IP对应的ID
				existingValue, found := b.GetKey(protocol.FilesLedgerKey, fileID)
				service := &types.Service{}
				existingValue.Unmarshal(service)
				// 如果不匹配，则更新区块链
				if !found || service.PeerID != n.Host().ID().String() {
					updatedMap := map[string]interface{}{}
					updatedMap[fileID] = types.File{PeerID: n.Host().ID().String(), Name: fileID}
					b.Add(protocol.FilesLedgerKey, updatedMap)
				}
			},
		)
		return nil
	}
}

// ShareFile 将文件共享到P2P网络。
// 应在节点使用Start()启动之前调用
// 参数 ll 为日志记录器，announcetime 为公告时间间隔，fileID 为文件ID，filepath 为文件路径
func ShareFile(ll log.StandardLogger, announcetime time.Duration, fileID, filepath string) ([]node.Option, error) {
	_, err := os.Stat(filepath)
	if err != nil {
		return nil, err
	}

	ll.Infof("正在提供 '%s' 作为 '%s'", filepath, fileID)
	return []node.Option{
		node.WithNetworkService(
			SharefileNetworkService(announcetime, fileID),
		),
		node.WithStreamHandler(protocol.FileProtocol,
			func(n *node.Node, l *blockchain.Ledger) func(stream network.Stream) {
				return func(stream network.Stream) {
					go func() {
						ll.Infof("(文件 %s) 收到来自 %s 的连接", fileID, stream.Conn().RemotePeer().String())

						// 从区块链中检索当前IP对应的ID
						_, found := l.GetKey(protocol.UsersLedgerKey, stream.Conn().RemotePeer().String())
						// 如果不匹配，则更新区块链
						if !found {
							ll.Info("重置", stream.Conn().RemotePeer().String(), "在账本中未找到")
							stream.Reset()
							return
						}
						f, err := os.Open(filepath)
						if err != nil {
							return
						}
						io.Copy(stream, f)
						f.Close()
						stream.Close()

						ll.Infof("(文件 %s) 完成处理 %s", fileID, stream.Conn().RemotePeer().String())
					}()
				}
			})}, nil

}

// ReceiveFile 接收文件
// 参数 ctx 为上下文，ledger 为区块链账本，n 为节点实例，l 为日志记录器，announcetime 为公告时间间隔，fileID 为文件ID，path 为保存路径
func ReceiveFile(ctx context.Context, ledger *blockchain.Ledger, n *node.Node, l log.StandardLogger, announcetime time.Duration, fileID string, path string) error {
	// 公告我们自己，以便节点接受我们的连接
	ledger.Announce(
		ctx,
		announcetime,
		func() {
			// 从区块链中检索当前IP对应的ID
			_, found := ledger.GetKey(protocol.UsersLedgerKey, n.Host().ID().String())

			// 如果不匹配，则更新区块链
			if !found {
				updatedMap := map[string]interface{}{}
				updatedMap[n.Host().ID().String()] = &types.User{
					PeerID:    n.Host().ID().String(),
					Timestamp: time.Now().String(),
				}
				ledger.Add(protocol.UsersLedgerKey, updatedMap)
			}
		},
	)

	for {
		select {
		case <-ctx.Done():
			return errors.New("上下文已取消")
		default:
			time.Sleep(5 * time.Second)

			l.Debug("尝试在区块链中查找文件")

			existingValue, found := ledger.GetKey(protocol.FilesLedgerKey, fileID)
			fi := &types.File{}
			existingValue.Unmarshal(fi)
			// 如果不匹配，则更新区块链
			if !found {
				l.Debug("文件在区块链中未找到，5秒后重试")
				continue
			} else {
				// 从区块链中检索当前IP对应的ID
				existingValue, found := ledger.GetKey(protocol.FilesLedgerKey, fileID)
				fi := &types.File{}
				existingValue.Unmarshal(fi)

				// 如果不匹配，则更新区块链
				if !found {
					return errors.New("文件未找到")
				}

				// 解码对等节点
				d, err := peer.Decode(fi.PeerID)
				if err != nil {
					return err
				}

				l.Debug("文件在区块链中找到，正在打开流到", d)

				// 打开流
				stream, err := n.Host().NewStream(ctx, d, protocol.FileProtocol.ID())
				if err != nil {
					l.Debugf("连接 %s 失败，5秒后重试", d)
					continue
				}

				l.Infof("正在保存文件 %s 到 %s", fileID, path)

				f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
				if err != nil {
					return err
				}

				io.Copy(f, stream)
				f.Close()

				l.Infof("已接收文件 %s 到 %s", fileID, path)
				return nil
			}
		}
	}
}
