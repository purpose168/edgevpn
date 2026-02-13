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

package hub

import "encoding/json"

// Message 消息结构体，在pubsub消息体中进行JSON转换
type Message struct {
	Message  string                 // 消息内容
	SenderID string                 // 发送者ID

	Annotations map[string]interface{} // 注解信息
}

// MessageOption 消息选项函数类型
type MessageOption func(cfg *Message) error

// Apply 应用给定的选项到配置，返回遇到的第一个错误（如果有）
func (m *Message) Apply(opts ...MessageOption) error {
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(m); err != nil {
			return err
		}
	}
	return nil
}

// NewMessage 创建新消息
// 参数 s 为消息内容
func NewMessage(s string) *Message {
	return &Message{Message: s}
}

// Copy 复制消息
func (m *Message) Copy() *Message {
	copy := *m
	return &copy
}

// WithMessage 设置消息内容并返回副本
// 参数 s 为新的消息内容
func (m *Message) WithMessage(s string) *Message {
	copy := m.Copy()
	copy.Message = s
	return copy
}

// AnnotationsToObj 将注解转换为对象
// 参数 v 为目标对象指针
func (m *Message) AnnotationsToObj(v interface{}) error {
	blob, err := json.Marshal(m.Annotations)
	if err != nil {
		return err
	}
	return json.Unmarshal(blob, v)
}
