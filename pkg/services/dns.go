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
	"fmt"
	"regexp"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/ipfs/go-log"
	"github.com/miekg/dns"
	"github.com/pkg/errors"
	"github.com/purpose168/edgevpn/pkg/blockchain"
	"github.com/purpose168/edgevpn/pkg/node"
	"github.com/purpose168/edgevpn/pkg/protocol"
	"github.com/purpose168/edgevpn/pkg/types"
)

// DNSNetworkService DNS网络服务
// 参数 ll 为日志记录器，listenAddr 为监听地址，forwarder 为是否启用转发，forward 为转发服务器列表，cacheSize 为缓存大小
func DNSNetworkService(ll log.StandardLogger, listenAddr string, forwarder bool, forward []string, cacheSize int) node.NetworkService {
	return func(ctx context.Context, c node.Config, n *node.Node, b *blockchain.Ledger) error {
		server := &dns.Server{Addr: listenAddr, Net: "udp"}
		cache, err := lru.New(cacheSize)
		if err != nil {
			return err
		}
		go func() {
			dns.HandleFunc(".", dnsHandler{ctx, b, forwarder, forward, cache, ll}.handleDNSRequest())
			fmt.Println(server.ListenAndServe())
		}()

		go func() {
			<-ctx.Done()
			server.Shutdown()
		}()

		return nil
	}
}

// DNS 返回在listenAddr上绑定DNS区块链解析器的网络服务。
// 接受区块链中地址的关联名称
// 参数 ll 为日志记录器，listenAddr 为监听地址，forwarder 为是否启用转发，forward 为转发服务器列表，cacheSize 为缓存大小
func DNS(ll log.StandardLogger, listenAddr string, forwarder bool, forward []string, cacheSize int) []node.Option {
	return []node.Option{
		node.WithNetworkService(DNSNetworkService(ll, listenAddr, forwarder, forward, cacheSize)),
	}
}

// PersistDNSRecord 是账本的语法糖
// 它将DNS记录持久化到区块链，直到看到它被协调。
// 它会自动停止公告，并且不*保证*持久化数据。
// 参数 ctx 为上下文，b 为区块链账本，announcetime 为公告时间，timeout 为超时时间，regex 为正则表达式，record 为DNS记录
func PersistDNSRecord(ctx context.Context, b *blockchain.Ledger, announcetime, timeout time.Duration, regex string, record types.DNS) {
	b.Persist(ctx, announcetime, timeout, protocol.DNSKey, regex, record)
}

// AnnounceDNSRecord 是账本的语法糖
// 将DNS记录绑定公告到区块链，并在ctx生命周期内持续公告
// 参数 ctx 为上下文，b 为区块链账本，announcetime 为公告时间，regex 为正则表达式，record 为DNS记录
func AnnounceDNSRecord(ctx context.Context, b *blockchain.Ledger, announcetime time.Duration, regex string, record types.DNS) {
	b.AnnounceUpdate(ctx, announcetime, protocol.DNSKey, regex, record)
}

// dnsHandler DNS处理器结构体
type dnsHandler struct {
	ctx       context.Context
	b         *blockchain.Ledger
	forwarder bool
	forward   []string
	cache     *lru.Cache
	ll        log.StandardLogger
}

// parseQuery 解析DNS查询
// 参数 m 为DNS消息，forward 为是否转发
func (d dnsHandler) parseQuery(m *dns.Msg, forward bool) *dns.Msg {
	response := m.Copy()
	d.ll.Debug("收到DNS请求", m)
	if len(m.Question) > 0 {
		q := m.Question[0]
		// 从区块链数据解析条目到IP
		for k, v := range d.b.CurrentData()[protocol.DNSKey] {
			r, err := regexp.Compile(k)
			if err == nil && r.MatchString(q.Name) {
				var res types.DNS
				v.Unmarshal(&res)
				if val, exists := res[dns.Type(q.Qtype)]; exists {
					rr, err := dns.NewRR(fmt.Sprintf("%s %s %s", q.Name, dns.TypeToString[q.Qtype], val))
					if err == nil {
						response.Answer = append(m.Answer, rr)
						d.ll.Debug("来自区块链的响应", response)
						return response
					}
				}
			}
		}
		if forward {
			d.ll.Debug("转发DNS请求", m)
			r, err := d.forwardQuery(m)
			if err == nil {
				response.Answer = r.Answer
			}
			d.ll.Debug("来自转发服务器的响应", r)
		}
	}
	return response
}

// handleDNSRequest 处理DNS请求
func (d dnsHandler) handleDNSRequest() func(w dns.ResponseWriter, r *dns.Msg) {
	return func(w dns.ResponseWriter, r *dns.Msg) {
		var resp *dns.Msg
		switch r.Opcode {
		case dns.OpcodeQuery:
			resp = d.parseQuery(r, d.forwarder)
		}
		resp.SetReply(r)
		resp.Compress = false
		w.WriteMsg(resp)
	}
}

// forwardQuery 转发DNS查询
// 参数 dnsMessage 为DNS消息
func (d dnsHandler) forwardQuery(dnsMessage *dns.Msg) (*dns.Msg, error) {
	reqCopy := dnsMessage.Copy()
	if len(reqCopy.Question) > 0 {
		if v, ok := d.cache.Get(reqCopy.Question[0].String()); ok {
			q := v.(*dns.Msg)
			q.Id = reqCopy.Id
			return q, nil
		}
	}
	for _, server := range d.forward {
		r, err := QueryDNS(d.ctx, reqCopy, server)
		if r != nil && len(r.Answer) == 0 && !r.MsgHdr.Truncated {
			continue
		}

		if err != nil {
			continue
		}

		if r.Rcode == dns.RcodeSuccess {
			d.cache.Add(reqCopy.Question[0].String(), r)
		}

		if r.Rcode == dns.RcodeNameError || r.Rcode == dns.RcodeSuccess || err == nil {
			return r, err
		}
	}
	return nil, errors.New("不可用")
}

// QueryDNS 使用DNS消息查询DNS服务器并返回答案
// 这是阻塞操作
// 参数 ctx 为上下文，msg 为DNS消息，dnsServer 为DNS服务器地址
func QueryDNS(ctx context.Context, msg *dns.Msg, dnsServer string) (*dns.Msg, error) {
	client := &dns.Client{
		Net:            "udp",
		Timeout:        30 * time.Second,
		SingleInflight: true}
	r, _, err := client.Exchange(msg, dnsServer)
	return r, err
}
