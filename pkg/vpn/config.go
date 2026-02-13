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

package vpn

import (
	"time"

	"github.com/ipfs/go-log"
	"github.com/mudler/water"
)

// Config VPN配置结构体，包含VPN接口和运行时参数
type Config struct {
	Interface        *water.Interface   // 网络接口实例
	InterfaceName    string             // 接口名称
	InterfaceAddress string             // 接口IP地址（CIDR格式）
	RouterAddress    string             // 路由器地址
	InterfaceMTU     int                // 接口MTU值
	MTU              int                // 数据包MTU值
	DeviceType       water.DeviceType   // 设备类型（TUN/TAP）

	LedgerAnnounceTime time.Duration    // 账本公告时间间隔
	Logger             log.StandardLogger // 日志记录器

	NetLinkBootstrap bool               // 是否使用NetLink引导

	// Frame timeout 帧超时时间
	Timeout time.Duration

	Concurrency       int               // 并发处理数
	ChannelBufferSize int               // 通道缓冲区大小
	MaxStreams        int               // 最大流数量
	lowProfile        bool              // 低配置模式
}

// Option 配置选项函数类型
type Option func(cfg *Config) error

// Apply 应用给定的选项到配置，返回遇到的第一个错误（如果有）
func (cfg *Config) Apply(opts ...Option) error {
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(cfg); err != nil {
			return err
		}
	}
	return nil
}

// WithMaxStreams 设置最大流数量的选项
func WithMaxStreams(i int) func(cfg *Config) error {
	return func(cfg *Config) error {
		cfg.MaxStreams = i
		return nil
	}
}

// LowProfile 低配置模式选项，启用后会限制资源使用
var LowProfile Option = func(cfg *Config) error {
	cfg.lowProfile = true

	return nil
}

// WithInterface 设置网络接口的选项
func WithInterface(i *water.Interface) func(cfg *Config) error {
	return func(cfg *Config) error {
		cfg.Interface = i
		return nil
	}
}

// NetLinkBootstrap 设置是否使用NetLink引导的选项
func NetLinkBootstrap(b bool) func(cfg *Config) error {
	return func(cfg *Config) error {
		cfg.NetLinkBootstrap = b
		return nil
	}
}

// WithTimeout 设置超时时间的选项
// 参数 s 为超时时间字符串（如 "15s"）
func WithTimeout(s string) Option {
	return func(cfg *Config) error {
		d, err := time.ParseDuration(s)
		cfg.Timeout = d
		return err
	}
}

// Logger 设置日志记录器的选项
func Logger(l log.StandardLogger) func(cfg *Config) error {
	return func(cfg *Config) error {
		cfg.Logger = l
		return nil
	}
}

// WithRouterAddress 设置路由器地址的选项
func WithRouterAddress(i string) func(cfg *Config) error {
	return func(cfg *Config) error {
		cfg.RouterAddress = i
		return nil
	}
}

// WithLedgerAnnounceTime 设置账本公告时间间隔的选项
func WithLedgerAnnounceTime(t time.Duration) func(cfg *Config) error {
	return func(cfg *Config) error {
		cfg.LedgerAnnounceTime = t
		return nil
	}
}

// WithConcurrency 设置并发处理数的选项
func WithConcurrency(i int) Option {
	return func(cfg *Config) error {
		cfg.Concurrency = i
		return nil
	}
}

// WithChannelBufferSize 设置通道缓冲区大小的选项
func WithChannelBufferSize(i int) Option {
	return func(cfg *Config) error {
		cfg.ChannelBufferSize = i
		return nil
	}
}

// WithInterfaceMTU 设置接口MTU值的选项
func WithInterfaceMTU(i int) func(cfg *Config) error {
	return func(cfg *Config) error {
		cfg.InterfaceMTU = i
		return nil
	}
}

// WithPacketMTU 设置数据包MTU值的选项
func WithPacketMTU(i int) func(cfg *Config) error {
	return func(cfg *Config) error {
		cfg.MTU = i
		return nil
	}
}

// WithInterfaceType 设置接口设备类型的选项
func WithInterfaceType(d water.DeviceType) func(cfg *Config) error {
	return func(cfg *Config) error {
		cfg.DeviceType = d
		return nil
	}
}

// WithInterfaceName 设置接口名称的选项
func WithInterfaceName(i string) func(cfg *Config) error {
	return func(cfg *Config) error {
		cfg.InterfaceName = i
		return nil
	}
}

// WithInterfaceAddress 设置接口IP地址的选项
func WithInterfaceAddress(i string) func(cfg *Config) error {
	return func(cfg *Config) error {
		cfg.InterfaceAddress = i
		return nil
	}
}
