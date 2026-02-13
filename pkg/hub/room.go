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

package hub

import (
	"context"
	"encoding/json"

	"github.com/libp2p/go-libp2p/core/peer"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

// room 表示对单个PubSub主题的订阅。
// 可以使用Room.Publish向主题发布消息，
// 接收到的消息被推送到Messages通道。
type room struct {
	ctx   context.Context
	ps    *pubsub.PubSub
	Topic *pubsub.Topic
	sub   *pubsub.Subscription

	roomName string  // 房间名称
	self     peer.ID // 自身对等节点ID
}

// connect 尝试订阅房间名称的PubSub主题，成功时返回Room
// 参数 ctx 为上下文，ps 为PubSub服务，selfID 为自身ID，roomName 为房间名称，messageChan 为消息通道
func connect(ctx context.Context, ps *pubsub.PubSub, selfID peer.ID, roomName string, messageChan chan *Message) (*room, error) {
	// 加入pubsub主题
	topic, err := ps.Join(roomName)
	if err != nil {
		return nil, err
	}

	// 订阅主题
	sub, err := topic.Subscribe()
	if err != nil {
		return nil, err
	}

	cr := &room{
		ctx:      ctx,
		ps:       ps,
		Topic:    topic,
		sub:      sub,
		self:     selfID,
		roomName: roomName,
	}

	// 在循环中开始从订阅读取消息
	go cr.readLoop(messageChan)
	return cr, nil
}

// publishMessage 向pubsub主题发送消息
// 参数 m 为要发布的消息
func (cr *room) publishMessage(m *Message) error {
	msgBytes, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return cr.Topic.Publish(cr.ctx, msgBytes)
}

// readLoop 从pubsub主题拉取消息并推送到Messages通道
// 参数 messageChan 为消息通道
func (cr *room) readLoop(messageChan chan *Message) {
	for {
		msg, err := cr.sub.Next(cr.ctx)
		if err != nil {
			return
		}
		// 只转发由其他人发送的消息
		if msg.ReceivedFrom == cr.self {
			continue
		}
		cm := new(Message)
		err = json.Unmarshal(msg.Data, cm)
		if err != nil {
			continue
		}

		cm.SenderID = msg.ReceivedFrom.String()

		// 将有效消息发送到Messages通道
		messageChan <- cm
	}
}
