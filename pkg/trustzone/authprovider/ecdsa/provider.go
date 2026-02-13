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

package ecdsa

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/ipfs/go-log/v2"
	"github.com/purpose168/edgevpn/pkg/blockchain"
	"github.com/purpose168/edgevpn/pkg/hub"
	"github.com/purpose168/edgevpn/pkg/node"
)

// ECDSA521 ECDSA521认证提供者结构体
type ECDSA521 struct {
	privkey string             // 私钥
	logger  log.StandardLogger // 日志记录器
}

// ECDSA521Provider 返回一个ECDSA521认证提供者。
// 使用时，请使用以下配置提供私钥：
// AuthProviders: map[string]map[string]interface{}{"ecdsa": {"private_key": "<key>"}},
// 运行时，也可以从TZ节点通过API添加密钥，例如：
// curl -X PUT 'http://localhost:8081/api/ledger/trustzoneAuth/ecdsa_1/<key>'
// 注意：privkey和pubkeys的格式由下面的GenerateKeys()生成
// 该提供者解析信任区域认证区域中的"ecdsa"密钥，
// 并使用每个密钥作为公钥尝试进行认证
// 参数 ll 为日志记录器，privkey 为私钥字符串
func ECDSA521Provider(ll log.StandardLogger, privkey string) (*ECDSA521, error) {
	return &ECDSA521{privkey: privkey, logger: ll}, nil
}

// Authenticate 根据一组公钥认证消息。
// 它遍历所有信任区域认证数据（提供者选项，不是存储发送者ID的地方）
// 并检测任何带有ecdsa前缀的密钥。值假定为字符串并解析为公钥。
// 然后使用公钥认证节点并验证是否有任何公钥验证了挑战。
// 参数 m 为消息，c 为消息通道，tzdata 为信任区域数据
func (e *ECDSA521) Authenticate(m *hub.Message, c chan *hub.Message, tzdata map[string]blockchain.Data) bool {

	// 获取消息签名
	sigs, ok := m.Annotations["sigs"]
	if !ok {
		e.logger.Debug("消息中没有签名", m.Message, m.Annotations)

		return false
	}

	e.logger.Debug("ECDSA认证收到", m)

	// 收集所有公钥
	pubKeys := []string{}
	for k, t := range tzdata {
		if strings.Contains(k, "ecdsa") {
			var s string
			t.Unmarshal(&s)
			pubKeys = append(pubKeys, s)
		}
	}
	if len(pubKeys) == 0 {
		e.logger.Debug("ECDSA认证：没有可认证的公钥")
		// 账本中没有可认证的公钥
		return false
	}
	for _, pubkey := range pubKeys {
		// 尝试验证签名
		if err := verify([]byte(pubkey), []byte(fmt.Sprint(sigs)), bytes.NewBufferString(m.Message)); err == nil {
			e.logger.Debug("ECDSA认证：签名已验证")
			return true
		}
		e.logger.Debug("ECDSA认证：签名未验证")
	}
	return false
}

// Challenger 如果当前节点不在信任区域中，则在公共通道上发送ECDSA521挑战。
// 这会启动一个挑战，最终应该让节点进入TZ
// 参数 inTrustZone 表示是否在信任区域中，c 为节点配置，n 为节点实例，b 为账本，trustData 为信任数据
func (e *ECDSA521) Challenger(inTrustZone bool, c node.Config, n *node.Node, b *blockchain.Ledger, trustData map[string]blockchain.Data) {
	if !inTrustZone {
		e.logger.Debug("ECDSA认证：当前节点不在信任区域中，发送挑战")
		signature, err := sign([]byte(e.privkey), bytes.NewBufferString("challenge"))
		if err != nil {
			e.logger.Error("签名消息错误：", err.Error())
			return
		}
		msg := hub.NewMessage("challenge")
		msg.Annotations = make(map[string]interface{})
		msg.Annotations["sigs"] = string(signature)
		n.PublishMessage(msg)
		return
	}
}
