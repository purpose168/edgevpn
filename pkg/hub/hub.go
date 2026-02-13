// Copyright © 2022 Ettore Di Giacinto <mudler@c3os.io>
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

package hub

import (
	"context"
	"crypto/sha256"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/purpose168/edgevpn/pkg/crypto"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

// MessageHub 消息中心结构体
type MessageHub struct {
	sync.Mutex

	blockchain, public *room          // 区块链房间和公共房间
	ps                 *pubsub.PubSub // 发布订阅服务
	otpKey             string         // OTP密钥
	maxsize            int            // 最大消息大小
	keyLength          int            // 密钥长度
	interval           int            // 间隔
	joinPublic         bool           // 是否加入公共房间

	ctxCancel                context.CancelFunc // 上下文取消函数
	Messages, PublicMessages chan *Message      // 消息通道和公共消息通道
}

// roomBufSize 是每个主题缓冲的传入消息数量
const roomBufSize = 128

// NewHub 创建新的消息中心
// 参数 otp 为OTP密钥，maxsize 为最大消息大小，keyLength 为密钥长度，interval 为间隔，joinPublic 为是否加入公共房间
func NewHub(otp string, maxsize, keyLength, interval int, joinPublic bool) *MessageHub {
	return &MessageHub{otpKey: otp, maxsize: maxsize, keyLength: keyLength, interval: interval,
		Messages: make(chan *Message, roomBufSize), PublicMessages: make(chan *Message, roomBufSize), joinPublic: joinPublic}
}

// topicKey 生成主题密钥
// 参数 salts 为可选的盐值
func (m *MessageHub) topicKey(salts ...string) string {
	totp := crypto.TOTP(sha256.New, m.keyLength, m.interval, m.otpKey)
	if len(salts) > 0 {
		return crypto.MD5(totp + strings.Join(salts, ":"))
	}
	return crypto.MD5(totp)
}

// joinRoom 加入房间
// 参数 host 为libp2p主机
func (m *MessageHub) joinRoom(host host.Host) error {
	m.Lock()
	defer m.Unlock()

	// 取消之前的上下文
	if m.ctxCancel != nil {
		m.ctxCancel()
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.ctxCancel = cancel

	// 使用GossipSub路由器创建新的PubSub服务
	ps, err := pubsub.NewGossipSub(ctx, host, pubsub.WithMaxMessageSize(m.maxsize))
	if err != nil {
		return err
	}

	// 加入"聊天"房间
	cr, err := connect(ctx, ps, host.ID(), m.topicKey(), m.Messages)
	if err != nil {
		return err
	}

	m.blockchain = cr

	// 如果启用公共房间，也加入公共房间
	if m.joinPublic {
		cr2, err := connect(ctx, ps, host.ID(), m.topicKey("public"), m.PublicMessages)
		if err != nil {
			return err
		}
		m.public = cr2
	}

	m.ps = ps

	return nil
}

// Start 启动消息中心
// 参数 ctx 为上下文，host 为libp2p主机
func (m *MessageHub) Start(ctx context.Context, host host.Host) error {
	c := make(chan interface{})
	go func(c context.Context, cc chan interface{}) {
		k := ""
		for {
			select {
			default:
				currentKey := m.topicKey()
				if currentKey != k {
					k = currentKey
					cc <- nil
				}
				time.Sleep(1 * time.Second)
			case <-ctx.Done():
				close(cc)
				return
			}
		}
	}(ctx, c)

	for range c {
		m.joinRoom(host)
	}

	// 关闭可能打开的上下文
	if m.ctxCancel != nil {
		m.ctxCancel()
	}
	return nil
}

// PublishMessage 发布消息到区块链房间
// 参数 mess 为要发布的消息
func (m *MessageHub) PublishMessage(mess *Message) error {
	m.Lock()
	defer m.Unlock()
	if m.blockchain != nil {
		return m.blockchain.publishMessage(mess)
	}
	return errors.New("没有可用的消息房间")
}

// PublishPublicMessage 发布消息到公共房间
// 参数 mess 为要发布的消息
func (m *MessageHub) PublishPublicMessage(mess *Message) error {
	m.Lock()
	defer m.Unlock()
	if m.public != nil {
		return m.public.publishMessage(mess)
	}
	return errors.New("没有可用的消息房间")
}

// ListPeers 列出房间中的对等节点
func (m *MessageHub) ListPeers() ([]peer.ID, error) {
	m.Lock()
	defer m.Unlock()
	if m.blockchain != nil {
		return m.blockchain.Topic.ListPeers(), nil
	}
	return nil, errors.New("没有可用的消息房间")
}
