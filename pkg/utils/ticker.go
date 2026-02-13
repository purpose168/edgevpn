package utils

import (
	"time"

	backoff "github.com/cenkalti/backoff/v4"
)

// expBackoffOpt 指数退避选项函数类型
type expBackoffOpt func(e *backoff.ExponentialBackOff)

// BackoffInitialInterval 设置初始间隔时间的选项
// 参数 i 为初始间隔时间
func BackoffInitialInterval(i time.Duration) expBackoffOpt {
	return func(e *backoff.ExponentialBackOff) {
		e.InitialInterval = i
	}
}

// BackoffRandomizationFactor 设置随机化因子的选项
// 参数 i 为随机化因子（0-1之间）
func BackoffRandomizationFactor(i float64) expBackoffOpt {
	return func(e *backoff.ExponentialBackOff) {
		e.RandomizationFactor = i
	}
}

// BackoffMultiplier 设置乘数的选项
// 参数 i 为乘数值
func BackoffMultiplier(i float64) expBackoffOpt {
	return func(e *backoff.ExponentialBackOff) {
		e.Multiplier = i
	}
}

// BackoffMaxInterval 设置最大间隔时间的选项
// 参数 i 为最大间隔时间
func BackoffMaxInterval(i time.Duration) expBackoffOpt {
	return func(e *backoff.ExponentialBackOff) {
		e.MaxInterval = i
	}
}

// BackoffMaxElapsedTime 设置最大已用时间的选项
// 参数 i 为最大已用时间
func BackoffMaxElapsedTime(i time.Duration) expBackoffOpt {
	return func(e *backoff.ExponentialBackOff) {
		e.MaxElapsedTime = i
	}
}

// newExpBackoff 创建新的指数退避实例
// 参数 o 为可选的配置选项
// 返回配置好的退避策略实例
func newExpBackoff(o ...expBackoffOpt) backoff.BackOff {
	b := &backoff.ExponentialBackOff{
		InitialInterval:     5 * time.Second,     // 初始间隔时间
		RandomizationFactor: 0.5,                 // 随机化因子
		Multiplier:          2,                   // 乘数
		MaxInterval:         2 * time.Minute,     // 最大间隔时间
		MaxElapsedTime:      0,                   // 最大已用时间（0表示无限制）
		Stop:                backoff.Stop,        // 停止标志
		Clock:               backoff.SystemClock, // 系统时钟
	}
	// 应用所有选项
	for _, opt := range o {
		opt(b)
	}
	b.Reset()
	return b
}

// NewBackoffTicker 创建新的退避定时器
// 参数 o 为可选的配置选项
// 返回退避定时器实例
func NewBackoffTicker(o ...expBackoffOpt) *backoff.Ticker {
	return backoff.NewTicker(newExpBackoff(o...))
}
