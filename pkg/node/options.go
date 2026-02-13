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

package node

import (
	"encoding/base64"
	"io/ioutil"
	"time"

	"github.com/ipfs/go-log"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"
	"github.com/purpose168/edgevpn/pkg/blockchain"
	discovery "github.com/purpose168/edgevpn/pkg/discovery"
	"github.com/purpose168/edgevpn/pkg/protocol"
	"github.com/purpose168/edgevpn/pkg/utils"
	"gopkg.in/yaml.v2"
)

// WithLibp2pOptions 覆盖默认选项
func WithLibp2pOptions(i ...libp2p.Option) func(cfg *Config) error {
	return func(cfg *Config) error {
		cfg.Options = i
		return nil
	}
}

// WithSealer 设置密封器
func WithSealer(i Sealer) Option {
	return func(cfg *Config) error {
		cfg.Sealer = i
		return nil
	}
}

// WithLibp2pAdditionalOptions 添加额外的libp2p选项
func WithLibp2pAdditionalOptions(i ...libp2p.Option) func(cfg *Config) error {
	return func(cfg *Config) error {
		cfg.AdditionalOptions = append(cfg.AdditionalOptions, i...)
		return nil
	}
}

// WithNetworkService 添加网络服务
func WithNetworkService(ns ...NetworkService) func(cfg *Config) error {
	return func(cfg *Config) error {
		cfg.NetworkServices = append(cfg.NetworkServices, ns...)
		return nil
	}
}

// WithInterfaceAddress 设置接口地址
func WithInterfaceAddress(i string) func(cfg *Config) error {
	return func(cfg *Config) error {
		cfg.InterfaceAddress = i
		return nil
	}
}

// WithBlacklist 设置黑名单
func WithBlacklist(i ...string) func(cfg *Config) error {
	return func(cfg *Config) error {
		cfg.Blacklist = i
		return nil
	}
}

// Logger 设置日志记录器
func Logger(l log.StandardLogger) func(cfg *Config) error {
	return func(cfg *Config) error {
		cfg.Logger = l
		return nil
	}
}

// WithStore 设置存储器
func WithStore(s blockchain.Store) func(cfg *Config) error {
	return func(cfg *Config) error {
		cfg.Store = s
		return nil
	}
}

// Handlers 添加处理器到列表，每个接收的消息都会调用
func Handlers(h ...Handler) func(cfg *Config) error {
	return func(cfg *Config) error {
		cfg.Handlers = append(cfg.Handlers, h...)
		return nil
	}
}

// GenericChannelHandlers 添加处理器到列表，在通用通道中每个接收的消息都会调用（不是为区块链分配的通道）
func GenericChannelHandlers(h ...Handler) func(cfg *Config) error {
	return func(cfg *Config) error {
		cfg.GenericChannelHandler = append(cfg.GenericChannelHandler, h...)
		return nil
	}
}

// WithStreamHandler 添加流处理器到列表，每个接收的消息都会调用
func WithStreamHandler(id protocol.Protocol, h StreamHandler) func(cfg *Config) error {
	return func(cfg *Config) error {
		cfg.StreamHandlers[id] = h
		return nil
	}
}

// DiscoveryService 将给定的服务添加到发现服务
func DiscoveryService(s ...ServiceDiscovery) func(cfg *Config) error {
	return func(cfg *Config) error {
		cfg.ServiceDiscovery = append(cfg.ServiceDiscovery, s...)
		return nil
	}
}

// EnableGenericHub 启用对等节点之间的额外通用中心
// 这可用于在对等节点之间交换与任何区块链事件无关的消息
// 例如，消息可用于认证或其他类型的应用程序
var EnableGenericHub = func(cfg *Config) error {
	cfg.GenericHub = true
	return nil
}

// ListenAddresses 设置监听地址
func ListenAddresses(ss ...string) func(cfg *Config) error {
	return func(cfg *Config) error {
		for _, s := range ss {
			a := &discovery.AddrList{}
			err := a.Set(s)
			if err != nil {
				return err
			}
			cfg.ListenAddresses = append(cfg.ListenAddresses, *a)
		}
		return nil
	}
}

// Insecure 设置是否禁用安全传输
func Insecure(b bool) func(cfg *Config) error {
	return func(cfg *Config) error {
		cfg.Insecure = b
		return nil
	}
}

// ExchangeKeys 设置交换密钥
func ExchangeKeys(s string) func(cfg *Config) error {
	return func(cfg *Config) error {
		cfg.ExchangeKey = s
		return nil
	}
}

// RoomName 设置房间名称
func RoomName(s string) func(cfg *Config) error {
	return func(cfg *Config) error {
		cfg.RoomName = s
		return nil
	}
}

// SealKeyInterval 设置密封密钥间隔
func SealKeyInterval(i int) func(cfg *Config) error {
	return func(cfg *Config) error {
		cfg.SealKeyInterval = i
		return nil
	}
}

// SealKeyLength 设置密封密钥长度
func SealKeyLength(i int) func(cfg *Config) error {
	return func(cfg *Config) error {
		cfg.SealKeyLength = i
		return nil
	}
}

// LibP2PLogLevel 设置libp2p日志级别
func LibP2PLogLevel(l log.LogLevel) func(cfg *Config) error {
	return func(cfg *Config) error {
		log.SetAllLoggers(l)
		return nil
	}
}

// MaxMessageSize 设置最大消息大小
func MaxMessageSize(i int) func(cfg *Config) error {
	return func(cfg *Config) error {
		cfg.MaxMessageSize = i
		return nil
	}
}

// WithPeerGater 设置对等节点门控器
func WithPeerGater(d Gater) Option {
	return func(cfg *Config) error {
		cfg.PeerGater = d
		return nil
	}
}

// WithLedgerAnnounceTime 设置账本公告时间
func WithLedgerAnnounceTime(t time.Duration) func(cfg *Config) error {
	return func(cfg *Config) error {
		cfg.LedgerAnnounceTime = t
		return nil
	}
}

// WithLedgerInterval 设置账本同步间隔
func WithLedgerInterval(t time.Duration) func(cfg *Config) error {
	return func(cfg *Config) error {
		cfg.LedgerSyncronizationTime = t
		return nil
	}
}

// WithDiscoveryInterval 设置发现间隔
func WithDiscoveryInterval(t time.Duration) func(cfg *Config) error {
	return func(cfg *Config) error {
		cfg.DiscoveryInterval = t
		return nil
	}
}

// WithDiscoveryBootstrapPeers 设置发现引导节点
func WithDiscoveryBootstrapPeers(a discovery.AddrList) func(cfg *Config) error {
	return func(cfg *Config) error {
		cfg.DiscoveryBootstrapPeers = a
		return nil
	}
}

// WithPrivKey 设置私钥
func WithPrivKey(b []byte) func(cfg *Config) error {
	return func(cfg *Config) error {
		cfg.PrivateKey = b
		return nil
	}
}

// WithStaticPeer 设置静态对等节点
func WithStaticPeer(ip string, p peer.ID) func(cfg *Config) error {
	return func(cfg *Config) error {
		if cfg.PeerTable == nil {
			cfg.PeerTable = make(map[string]peer.ID)
		}
		cfg.PeerTable[ip] = p
		return nil
	}
}

// OTPConfig OTP配置结构体
type OTPConfig struct {
	Interval int    `yaml:"interval"` // 间隔
	Key      string `yaml:"key"`      // 密钥
	Length   int    `yaml:"length"`   // 长度
}

// OTP OTP配置
type OTP struct {
	DHT    OTPConfig `yaml:"dht"`    // DHT配置
	Crypto OTPConfig `yaml:"crypto"` // 加密配置
}

// YAMLConnectionConfig YAML连接配置结构体
type YAMLConnectionConfig struct {
	OTP OTP `yaml:"otp"`

	RoomName       string `yaml:"room"`             // 房间名称
	Rendezvous     string `yaml:"rendezvous"`       // 会合点
	MDNS           string `yaml:"mdns"`             // mDNS
	MaxMessageSize int    `yaml:"max_message_size"` // 最大消息大小
}

// Base64 返回连接配置的base64字符串表示
func (y YAMLConnectionConfig) Base64() string {
	bytesData, _ := yaml.Marshal(y)
	return base64.StdEncoding.EncodeToString(bytesData)
}

// YAML 返回连接配置的YAML字符串
func (y YAMLConnectionConfig) YAML() string {
	bytesData, _ := yaml.Marshal(y)
	return string(bytesData)
}

// copy 复制配置到目标配置
func (y YAMLConnectionConfig) copy(mdns, dht bool, cfg *Config, d *discovery.DHT, m *discovery.MDNS) {
	if d == nil {
		d = discovery.NewDHT()
	}
	if m == nil {
		m = &discovery.MDNS{}
	}

	d.RefreshDiscoveryTime = cfg.DiscoveryInterval
	d.OTPInterval = y.OTP.DHT.Interval
	d.OTPKey = y.OTP.DHT.Key
	d.KeyLength = y.OTP.DHT.Length
	d.RendezvousString = y.Rendezvous
	d.BootstrapPeers = cfg.DiscoveryBootstrapPeers

	m.DiscoveryServiceTag = y.MDNS
	cfg.ExchangeKey = y.OTP.Crypto.Key
	cfg.RoomName = y.RoomName
	cfg.SealKeyInterval = y.OTP.Crypto.Interval
	//	cfg.ServiceDiscovery = []ServiceDiscovery{d, m}
	if mdns {
		cfg.ServiceDiscovery = append(cfg.ServiceDiscovery, m)
	}
	if dht {
		cfg.ServiceDiscovery = append(cfg.ServiceDiscovery, d)
	}
	cfg.SealKeyLength = y.OTP.Crypto.Length
	cfg.MaxMessageSize = y.MaxMessageSize
}

// defaultKeyLength 默认密钥长度
const defaultKeyLength = 43

// GenerateNewConnectionData 生成新的连接数据
// 参数 i 为可选参数：间隔时间、最大消息大小、密钥长度
func GenerateNewConnectionData(i ...int) *YAMLConnectionConfig {
	defaultInterval := 9000
	maxMessSize := 20 << 20 // 20MB
	keyLength := defaultKeyLength

	if len(i) >= 3 {
		keyLength = i[2]
		defaultInterval = i[0]
		maxMessSize = i[1]
	} else if len(i) >= 2 {
		defaultInterval = i[0]
		maxMessSize = i[1]
	} else if len(i) == 1 {
		defaultInterval = i[0]
	}

	return &YAMLConnectionConfig{
		MaxMessageSize: maxMessSize,
		RoomName:       utils.RandStringRunes(keyLength),
		Rendezvous:     utils.RandStringRunes(keyLength),
		MDNS:           utils.RandStringRunes(keyLength),
		OTP: OTP{
			DHT: OTPConfig{
				Key:      utils.RandStringRunes(keyLength),
				Interval: defaultInterval,
				Length:   defaultKeyLength,
			},
			Crypto: OTPConfig{
				Key:      utils.RandStringRunes(keyLength),
				Interval: defaultInterval,
				Length:   defaultKeyLength,
			},
		},
	}
}

// FromYaml 从YAML文件加载配置
// 参数 enablemDNS 为是否启用mDNS，enableDHT 为是否启用DHT，path 为文件路径，d 为DHT发现，m 为mDNS发现
func FromYaml(enablemDNS, enableDHT bool, path string, d *discovery.DHT, m *discovery.MDNS) func(cfg *Config) error {
	return func(cfg *Config) error {
		if len(path) == 0 {
			return nil
		}
		t := YAMLConnectionConfig{}

		data, err := ioutil.ReadFile(path)
		if err != nil {
			return errors.Wrap(err, "读取yaml文件")
		}

		if err := yaml.Unmarshal(data, &t); err != nil {
			return errors.Wrap(err, "解析yaml")
		}

		t.copy(enablemDNS, enableDHT, cfg, d, m)
		return nil
	}
}

// FromBase64 从base64字符串加载配置
// 参数 enablemDNS 为是否启用mDNS，enableDHT 为是否启用DHT，bb 为base64字符串，d 为DHT发现，m 为mDNS发现
func FromBase64(enablemDNS, enableDHT bool, bb string, d *discovery.DHT, m *discovery.MDNS) func(cfg *Config) error {
	return func(cfg *Config) error {
		if len(bb) == 0 {
			return nil
		}
		configDec, err := base64.StdEncoding.DecodeString(bb)
		if err != nil {
			return err
		}
		t := YAMLConnectionConfig{}

		if err := yaml.Unmarshal(configDec, &t); err != nil {
			return errors.Wrap(err, "解析yaml")
		}
		t.copy(enablemDNS, enableDHT, cfg, d, m)
		return nil
	}
}
