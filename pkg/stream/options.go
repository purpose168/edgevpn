// The MIT License (MIT)

// Copyright (c) 2017 Whyrusleeping

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

// This package is a port of go-libp2p-connmgr, but adapted for streams
// 本包是go-libp2p-connmgr的移植版本，但针对流进行了适配

package stream

import (
	"errors"
	"time"
)

// config 是基本连接管理器的配置结构体
type config struct {
	highWater     int           // 高水位线
	lowWater      int           // 低水位线
	gracePeriod   time.Duration // 宽限期
	silencePeriod time.Duration // 静默期
	decayer       *DecayerCfg   // 衰减器配置
}

// Option 表示基本连接管理器的选项
type Option func(*config) error

// DecayerConfig 应用衰减器的配置
func DecayerConfig(opts *DecayerCfg) Option {
	return func(cfg *config) error {
		cfg.decayer = opts
		return nil
	}
}

// WithGracePeriod 设置宽限期
// 宽限期是新打开的连接在被修剪之前获得的时间
// 参数 p 为宽限期时长
func WithGracePeriod(p time.Duration) Option {
	return func(cfg *config) error {
		if p < 0 {
			return errors.New("宽限期必须为非负数")
		}
		cfg.gracePeriod = p
		return nil
	}
}

// WithSilencePeriod 设置静默期
// 如果连接数超过高水位线，连接管理器将在每个静默期执行一次清理
// 参数 p 为静默期时长
func WithSilencePeriod(p time.Duration) Option {
	return func(cfg *config) error {
		if p <= 0 {
			return errors.New("静默期必须为非零值")
		}
		cfg.silencePeriod = p
		return nil
	}
}
