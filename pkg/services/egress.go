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
	"bufio"
	"context"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/purpose168/edgevpn/pkg/blockchain"
	"github.com/purpose168/edgevpn/pkg/node"
	"github.com/purpose168/edgevpn/pkg/protocol"
	"github.com/purpose168/edgevpn/pkg/types"
)

// egressHandler 出口处理器
// 参数 n 为节点实例，b 为区块链账本
func egressHandler(n *node.Node, b *blockchain.Ledger) func(stream network.Stream) {
	return func(stream network.Stream) {
		// 记得在完成后关闭流
		defer stream.Close()

		// 从区块链中检索IP对应的当前ID
		_, found := b.GetKey(protocol.UsersLedgerKey, stream.Conn().RemotePeer().String())
		// 如果不匹配，则更新区块链
		if !found {
			//		ll.Debugf("重置 '%s': 在账本中未找到", stream.Conn().RemotePeer().String())
			stream.Reset()
			return
		}

		// 创建新的缓冲读取器，因为ReadRequest需要一个
		// 缓冲读取器从我们的流中读取，我们在流上
		// 发送了HTTP请求（参见ServeHTTP()）
		buf := bufio.NewReader(stream)
		// 从缓冲区读取HTTP请求
		req, err := http.ReadRequest(buf)
		if err != nil {
			stream.Reset()
			log.Println(err)
			return
		}
		defer req.Body.Close()

		// 我们需要重置请求中的这些字段
		// URL，因为它们没有被维护
		req.URL.Scheme = "http"
		hp := strings.Split(req.Host, ":")
		if len(hp) > 1 && hp[1] == "443" {
			req.URL.Scheme = "https"
		} else {
			req.URL.Scheme = "http"
		}
		req.URL.Host = req.Host

		outreq := new(http.Request)
		*outreq = *req

		// 现在我们发起请求
		//fmt.Printf("正在请求 %s\n", req.URL)
		resp, err := http.DefaultTransport.RoundTrip(outreq)
		if err != nil {
			stream.Reset()
			log.Println(err)
			return
		}

		// resp.Write 将我们为请求获得的任何响应
		// 写回流中
		resp.Write(stream)
	}
}

// ProxyService 启动本地HTTP代理服务器，将请求重定向到网络中的出口
// 参数 deadtime 用于考虑在时间窗口内存活的主机
// 参数 announceTime 为公告时间间隔，listenAddr 为监听地址，deadtime 为失效时间
func ProxyService(announceTime time.Duration, listenAddr string, deadtime time.Duration) node.NetworkService {
	return func(ctx context.Context, c node.Config, n *node.Node, b *blockchain.Ledger) error {

		ps := &proxyService{
			host:       n,
			listenAddr: listenAddr,
			deadTime:   deadtime,
		}

		// 公告我们自己，以便节点接受我们的连接
		b.Announce(
			ctx,
			announceTime,
			func() {
				// 从区块链中检索IP对应的当前ID
				_, found := b.GetKey(protocol.UsersLedgerKey, n.Host().ID().String())
				// 如果不匹配，则更新区块链
				if !found {
					updatedMap := map[string]interface{}{}
					updatedMap[n.Host().ID().String()] = &types.User{
						PeerID:    n.Host().ID().String(),
						Timestamp: time.Now().String(),
					}
					b.Add(protocol.UsersLedgerKey, updatedMap)
				}
			},
		)

		go ps.Serve()
		return nil
	}
}

// proxyService 代理服务结构体
type proxyService struct {
	host       *node.Node
	listenAddr string
	deadTime   time.Duration
}

// Serve 启动代理服务
func (p *proxyService) Serve() error {
	return http.ListenAndServe(p.listenAddr, p)
}

// ServeHTTP 处理HTTP请求
func (p *proxyService) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	l, err := p.host.Ledger()
	if err != nil {
		//fmt.Printf("没有账本")
		return
	}

	egress := l.CurrentData()[protocol.EgressService]
	nodes := AvailableNodes(l, p.deadTime)

	availableEgresses := []string{}
	for _, n := range nodes {
		for e := range egress {
			if e == n {
				availableEgresses = append(availableEgresses, e)
			}
		}
	}

	chosen := availableEgresses[rand.Intn(len(availableEgresses)-1)]

	//fmt.Printf("代理请求 %s 到对等节点 %s\n", r.URL, chosen)
	// 我们需要将请求发送到远程libp2p对等节点，所以
	// 我们打开一个流
	stream, err := p.host.Host().NewStream(context.Background(), peer.ID(chosen), protocol.EgressProtocol.ID())
	// 如果发生错误，我们写入错误响应
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer stream.Close()

	// r.Write() 将HTTP请求写入流
	err = r.Write(stream)
	if err != nil {
		stream.Reset()
		log.Println(err)
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	// 现在我们读取从目标对等节点发送的响应
	buf := bufio.NewReader(stream)
	resp, err := http.ReadResponse(buf, r)
	if err != nil {
		stream.Reset()
		log.Println(err)
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	// 复制所有头部
	for k, v := range resp.Header {
		for _, s := range v {
			w.Header().Add(k, s)
		}
	}

	// 写入响应状态和头部
	w.WriteHeader(resp.StatusCode)

	// 最后复制主体
	io.Copy(w, resp.Body)
	resp.Body.Close()
}

// EgressService 出口服务
// 参数 announceTime 为公告时间间隔
func EgressService(announceTime time.Duration) node.NetworkService {
	return func(ctx context.Context, c node.Config, n *node.Node, b *blockchain.Ledger) error {
		b.AnnounceUpdate(ctx, announceTime, protocol.EgressService, n.Host().ID().String(), "ok")
		return nil
	}
}

// Egress 返回出口服务选项
// 参数 announceTime 为公告时间间隔
func Egress(announceTime time.Duration) []node.Option {
	return []node.Option{
		node.WithNetworkService(EgressService(announceTime)),
		node.WithStreamHandler(protocol.EgressProtocol, egressHandler),
	}
}

// Proxy 返回代理服务选项
// 参数 announceTime 为公告时间间隔，deadtime 为失效时间，listenAddr 为监听地址
func Proxy(announceTime, deadtime time.Duration, listenAddr string) []node.Option {
	return []node.Option{
		node.WithNetworkService(ProxyService(announceTime, listenAddr, deadtime)),
	}
}
