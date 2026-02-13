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

package node

import (
	hub "github.com/mudler/edgevpn/pkg/hub"
)

// messageWriter 是由节点返回的结构体，满足io.Writer接口
// 在底层中心上实现
// 写入消息写入器的所有内容都会排队到消息通道
// 由节点进行密封和处理
type messageWriter struct {
	input chan<- *hub.Message // 输入通道
	c     Config              // 配置
	mess  *hub.Message        // 消息
}

// Write 将字节切片写入消息通道
func (mw *messageWriter) Write(p []byte) (n int, err error) {
	return mw.Send(mw.mess.WithMessage(string(p)))
}

// Send 将消息发送到通道
func (mw *messageWriter) Send(copy *hub.Message) (n int, err error) {
	mw.input <- copy
	return len(copy.Message), nil
}
