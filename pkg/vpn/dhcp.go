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

package vpn

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/ipfs/go-log/v2"
	"github.com/mudler/edgevpn/pkg/crypto"
	"github.com/mudler/edgevpn/pkg/node"
	"github.com/mudler/edgevpn/pkg/protocol"
	"github.com/mudler/edgevpn/pkg/services"
	"github.com/mudler/edgevpn/pkg/types"
	"github.com/mudler/edgevpn/pkg/utils"

	"github.com/mudler/edgevpn/pkg/blockchain"
)

// checkDHCPLease 检查是否存在DHCP租约文件
// 参数 c 为节点配置，leasedir 为租约目录
// 返回租约的IP地址字符串，如果不存在则返回空字符串
func checkDHCPLease(c node.Config, leasedir string) string {
	// 如果存在则检索租约
	leaseFileName := crypto.MD5(fmt.Sprintf("%s-ek", c.ExchangeKey))
	leaseFile := filepath.Join(leasedir, leaseFileName)
	if _, err := os.Stat(leaseFile); err == nil {
		b, _ := ioutil.ReadFile(leaseFile)
		return string(b)
	}
	return ""
}

// contains 检查字符串切片中是否包含指定元素
// 参数 slice 为字符串切片，elem 为要查找的元素
func contains(slice []string, elem string) bool {
	for _, s := range slice {
		if elem == s {
			return true
		}
	}
	return false
}

// DHCPNetworkService 返回一个DHCP网络服务
// 参数 ip 为IP地址通道，l 为日志记录器，maxTime 为最大超时时间，leasedir 为租约目录，address 为基础地址
func DHCPNetworkService(ip chan string, l log.StandardLogger, maxTime time.Duration, leasedir string, address string) node.NetworkService {
	return func(ctx context.Context, c node.Config, n *node.Node, b *blockchain.Ledger) error {
		// 创建租约目录
		os.MkdirAll(leasedir, 0600)

		// 如果存在则检索租约
		var wantedIP = checkDHCPLease(c, leasedir)

		// 任何需要新IP的节点：
		//  1. 获取可用节点。从Machine中过滤掉没有IP的节点。
		//  2. 在它们中选择领导者。如果我们不是，则等待
		//  3. 如果我们是领导者，选择一个IP并使用该IP启动VPN
		for wantedIP == "" {
			time.Sleep(5 * time.Second)

			// 此网络服务是阻塞的，在VPN之前调用，因此需要在VPN之前注册
			nodes := services.AvailableNodes(b, maxTime)

			currentIPs := map[string]string{}
			ips := []string{}

			// 遍历账本中的机器信息，收集当前IP分配情况
			for _, t := range b.LastBlock().Storage[protocol.MachinesLedgerKey] {
				var m types.Machine
				t.Unmarshal(&m)
				currentIPs[m.PeerID] = m.Address

				l.Debugf("%s 使用 %s", m.PeerID, m.Address)
				ips = append(ips, m.Address)
			}

			// 找出没有IP的节点
			nodesWithNoIP := []string{}
			for _, nn := range nodes {
				if _, exists := currentIPs[nn]; !exists {
					nodesWithNoIP = append(nodesWithNoIP, nn)
				}
			}

			// 节点数量不足以确定IP
			if len(nodes) <= 1 {
				l.Debug("节点数量不足以确定IP，休眠中")
				continue
			}

			// 没有需要IP的节点
			if len(nodesWithNoIP) == 0 {
				l.Debug("没有需要IP的节点，等待IP公告，休眠中")
				continue
			}

			// 确定应该成为领导者的节点
			shouldBeLeader := utils.Leader(nodesWithNoIP)

			var lead string
			v, exists := b.GetKey("dhcp", "leader")
			if exists {
				v.Unmarshal(&lead)
			}

			// 如果我们不是应该成为的领导者，也不是当前领导者，则等待
			if shouldBeLeader != n.Host().ID().String() && lead != n.Host().ID().String() {
				c.Logger.Infof("<%s> 不是领导者，领导者是 '%s'，休眠中", n.Host().ID().String(), shouldBeLeader)
				continue
			}

			// 如果我们应该成为领导者，但还没有宣布或当前领导者不在需要IP的节点列表中
			if shouldBeLeader == n.Host().ID().String() && (lead == "" || !contains(nodesWithNoIP, lead)) {
				b.Persist(ctx, 5*time.Second, 15*time.Second, "dhcp", "leader", n.Host().ID().String())
				c.Logger.Info("宣布我们为领导者，退避中")
				continue
			}

			// 如果我们不是当前标记的领导者
			if lead != n.Host().ID().String() {
				c.Logger.Info("退避中，因为我们当前未被标记为领导者")
				time.Sleep(5 * time.Second)
				continue
			}

			l.Debug("没有IP的节点", nodesWithNoIP)
			// 我们是领导者
			l.Debug("从中选择IP", ips)

			// 获取下一个可用IP
			wantedIP = utils.NextIP(address, ips)
		}

		// 将租约保存到磁盘
		leaseFileName := crypto.MD5(fmt.Sprintf("%s-ek", c.ExchangeKey))
		leaseFile := filepath.Join(leasedir, leaseFileName)
		l.Debugf("将租约写入 '%s'", leaseFile)
		if err := ioutil.WriteFile(leaseFile, []byte(wantedIP), 0600); err != nil {
			l.Warn(err)
		}

		// 将IP传播到通道，在启动VPN时读取
		ip <- wantedIP

		// 从VPN限制连接
		return n.BlockSubnet(fmt.Sprintf("%s/24", wantedIP))
	}
}

// DHCP 返回一个DHCP网络服务。它需要Alive服务来确定可用节点。
// 可用节点用于确定哪些节点需要IP，当maxTime过期时，节点被标记为离线并不再考虑。
// 参数 l 为日志记录器，maxTime 为最大超时时间，leasedir 为租约目录，address 为基础地址
// 返回节点选项和VPN选项
func DHCP(l log.StandardLogger, maxTime time.Duration, leasedir string, address string) ([]node.Option, []Option) {
	ip := make(chan string, 1)
	return []node.Option{
			func(cfg *node.Config) error {
				// 如果存在则检索租约。在启动节点时由连接限制器消费
				lease := checkDHCPLease(*cfg, leasedir)
				if lease != "" {
					cfg.InterfaceAddress = fmt.Sprintf("%s/24", lease)
				}
				return nil
			},
			node.WithNetworkService(DHCPNetworkService(ip, l, maxTime, leasedir, address)),
		}, []Option{
			func(cfg *Config) error {
				// 启动VPN时读取IP
				cfg.InterfaceAddress = fmt.Sprintf("%s/24", <-ip)
				close(ip)
				l.Debug("收到IP", cfg.InterfaceAddress)
				return nil
			},
		}
}
