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
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/libp2p/go-libp2p/core/connmgr"
	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/benbjohnson/clock"
)

// DefaultResolution 是衰减跟踪器的默认分辨率
var DefaultResolution = 1 * time.Minute

// bumpCmd 表示一个增量命令
type bumpCmd struct {
	peer  peer.ID
	tag   *decayingTag
	delta int
}

// removeCmd 表示一个标签移除命令
type removeCmd struct {
	peer peer.ID
	tag  *decayingTag
}

// decayer 跟踪并管理所有衰减标签及其值
type decayer struct {
	cfg   *DecayerCfg
	mgr   *Manager
	clock clock.Clock // 用于测试

	tagsMu    sync.Mutex
	knownTags map[string]*decayingTag

	// lastTick 存储衰减器上次滴答的时间。由原子操作保护。
	lastTick atomic.Value

	// bumpTagCh 将增量命令排队等待循环处理
	bumpTagCh   chan bumpCmd
	removeTagCh chan removeCmd
	closeTagCh  chan *decayingTag

	// 闭包相关
	closeCh chan struct{}
	doneCh  chan struct{}
	err     error
}

var _ connmgr.Decayer = (*decayer)(nil)

// DecayerCfg 是衰减器的配置对象
type DecayerCfg struct {
	Resolution time.Duration // 分辨率
	Clock      clock.Clock   // 时钟（用于测试）
}

// WithDefaults 在此DecayerConfig实例上写入默认值，
// 并返回自身以支持链式调用。
//
//	cfg := (&DecayerCfg{}).WithDefaults()
//	cfg.Resolution = 30 * time.Second
//	t := NewDecayer(cfg, cm)
func (cfg *DecayerCfg) WithDefaults() *DecayerCfg {
	cfg.Resolution = DefaultResolution
	return cfg
}

// NewDecayer 创建新的衰减标签注册表
func NewDecayer(cfg *DecayerCfg, mgr *Manager) (*decayer, error) {
	// 如果配置中的Clock为nil，则使用真实时间
	if cfg.Clock == nil {
		cfg.Clock = clock.New()
	}

	d := &decayer{
		cfg:         cfg,
		mgr:         mgr,
		clock:       cfg.Clock,
		knownTags:   make(map[string]*decayingTag),
		bumpTagCh:   make(chan bumpCmd, 128),
		removeTagCh: make(chan removeCmd, 128),
		closeTagCh:  make(chan *decayingTag, 128),
		closeCh:     make(chan struct{}),
		doneCh:      make(chan struct{}),
	}

	d.lastTick.Store(d.clock.Now())

	// 启动处理
	go d.process()

	return d, nil
}

// RegisterDecayingTag 注册衰减标签
// 参数 name 为标签名称，interval 为衰减间隔，decayFn 为衰减函数，bumpFn 为增量函数
func (d *decayer) RegisterDecayingTag(name string, interval time.Duration, decayFn connmgr.DecayFn, bumpFn connmgr.BumpFn) (connmgr.DecayingTag, error) {
	d.tagsMu.Lock()
	defer d.tagsMu.Unlock()

	if _, ok := d.knownTags[name]; ok {
		return nil, fmt.Errorf("名为 %s 的衰减标签已存在", name)
	}

	if interval < d.cfg.Resolution {
		log.Warnf("标签 %s 的衰减间隔（%s）低于跟踪器的分辨率（%s）；已覆盖为分辨率",
			name, interval, d.cfg.Resolution)
		interval = d.cfg.Resolution
	}

	if interval%d.cfg.Resolution != 0 {
		log.Warnf("标签 %s 的衰减间隔（%s）不是跟踪器分辨率（%s）的倍数；可能会丢失一些精度",
			name, interval, d.cfg.Resolution)
	}

	lastTick := d.lastTick.Load().(time.Time)
	tag := &decayingTag{
		trkr:     d,
		name:     name,
		interval: interval,
		nextTick: lastTick.Add(interval),
		decayFn:  decayFn,
		bumpFn:   bumpFn,
	}

	d.knownTags[name] = tag
	return tag, nil
}

// Close 关闭衰减器。它是幂等的。
func (d *decayer) Close() error {
	select {
	case <-d.doneCh:
		return d.err
	default:
	}

	close(d.closeCh)
	<-d.doneCh
	return d.err
}

// process 是跟踪器的核心。它执行以下职责：
//
//  1. 管理衰减。
//  2. 应用分数增量。
//  3. 关闭时退出。
func (d *decayer) process() {
	defer close(d.doneCh)

	ticker := d.clock.Ticker(d.cfg.Resolution)
	defer ticker.Stop()

	var (
		bmp   bumpCmd
		now   time.Time
		visit = make(map[*decayingTag]struct{})
	)

	for {
		select {
		case now = <-ticker.C:
			d.lastTick.Store(now)

			d.tagsMu.Lock()
			for _, tag := range d.knownTags {
				if tag.nextTick.After(now) {
					// 跳过此标签
					continue
				}
				// 标记此标签需要在本轮更新
				visit[tag] = struct{}{}
			}
			d.tagsMu.Unlock()

			// 访问每个对等节点，衰减需要衰减的标签
			for _, s := range d.mgr.segments {
				s.Lock()

				// 进入包含对等节点的分段。处理每个对等节点
				for _, p := range s.peers {
					for tag, v := range p.decaying {
						if _, ok := visit[tag]; !ok {
							// 跳过此标签
							continue
						}

						// ~ 此值需要被访问 ~
						var delta int
						if after, rm := tag.decayFn(*v); rm {
							// 删除该值并继续处理下一个标签
							delta -= v.Value
							delete(p.decaying, tag)
						} else {
							// 累积增量并应用更改
							delta += after - v.Value
							v.Value, v.LastVisit = after, now
						}
						p.value += delta
					}
				}

				s.Unlock()
			}

			// 重置每个标签的下次访问轮次，并清空已访问集合
			for tag := range visit {
				tag.nextTick = tag.nextTick.Add(tag.interval)
				delete(visit, tag)
			}

		case bmp = <-d.bumpTagCh:
			var (
				now       = d.clock.Now()
				peer, tag = bmp.peer, bmp.tag
			)

			s := d.mgr.segments.get(peer)
			s.Lock()

			p := s.tagInfoFor(peer)
			v, ok := p.decaying[tag]
			if !ok {
				v = &connmgr.DecayingValue{
					Tag:       tag,
					Peer:      peer,
					LastVisit: now,
					Added:     now,
					Value:     0,
				}
				p.decaying[tag] = v
			}

			prev := v.Value
			v.Value, v.LastVisit = v.Tag.(*decayingTag).bumpFn(*v, bmp.delta), now
			p.value += v.Value - prev

			s.Unlock()

		case rm := <-d.removeTagCh:
			s := d.mgr.segments.get(rm.peer)
			s.Lock()

			p := s.tagInfoFor(rm.peer)
			v, ok := p.decaying[rm.tag]
			if !ok {
				s.Unlock()
				continue
			}
			p.value -= v.Value
			delete(p.decaying, rm.tag)
			s.Unlock()

		case t := <-d.closeTagCh:
			// 停止跟踪此标签
			d.tagsMu.Lock()
			delete(d.knownTags, t.name)
			d.tagsMu.Unlock()

			// 从connmgr中所有拥有此标签的对等节点移除该标签
			for _, s := range d.mgr.segments {
				// 访问所有分段，尝试从中存储的所有对等节点移除该标签
				s.Lock()
				for _, p := range s.peers {
					if dt, ok := p.decaying[t]; ok {
						// 减少peerInfo的值，并删除该标签
						p.value -= dt.Value
						delete(p.decaying, t)
					}
				}
				s.Unlock()
			}

		case <-d.closeCh:
			return
		}
	}
}

// decayingTag 表示一个衰减标签，具有关联的衰减间隔、衰减函数和增量函数
type decayingTag struct {
	trkr     *decayer
	name     string
	interval time.Duration
	nextTick time.Time
	decayFn  connmgr.DecayFn
	bumpFn   connmgr.BumpFn

	// closed 将此标签标记为已关闭，以便在关闭后如果被增量，我们可以返回错误
	// 0 = false; 1 = true; 由原子操作保护
	closed int32
}

var _ connmgr.DecayingTag = (*decayingTag)(nil)

// Name 返回标签名称
func (t *decayingTag) Name() string {
	return t.name
}

// Interval 返回衰减间隔
func (t *decayingTag) Interval() time.Duration {
	return t.interval
}

// Bump 为此对等节点增加标签值
// 参数 p 为对等节点ID，delta 为增量值
func (t *decayingTag) Bump(p peer.ID, delta int) error {
	if atomic.LoadInt32(&t.closed) == 1 {
		return fmt.Errorf("衰减标签 %s 已关闭；不再接受增量", t.name)
	}

	bmp := bumpCmd{peer: p, tag: t, delta: delta}

	select {
	case t.trkr.bumpTagCh <- bmp:
		return nil
	default:
		return fmt.Errorf(
			"无法为对等节点 %s 增加衰减标签，标签 %s，增量 %d；队列已满（长度=%d）",
			p.String(), t.name, delta, len(t.trkr.bumpTagCh))
	}
}

// Remove 移除此对等节点的标签
// 参数 p 为对等节点ID
func (t *decayingTag) Remove(p peer.ID) error {
	if atomic.LoadInt32(&t.closed) == 1 {
		return fmt.Errorf("衰减标签 %s 已关闭；不再接受移除操作", t.name)
	}

	rm := removeCmd{peer: p, tag: t}

	select {
	case t.trkr.removeTagCh <- rm:
		return nil
	default:
		return fmt.Errorf(
			"无法为对等节点 %s 移除衰减标签，标签 %s；队列已满（长度=%d）",
			p.String(), t.name, len(t.trkr.removeTagCh))
	}
}

// Close 关闭衰减标签
func (t *decayingTag) Close() error {
	if !atomic.CompareAndSwapInt32(&t.closed, 0, 1) {
		log.Warnf("重复关闭衰减标签: %s；跳过", t.name)
		return nil
	}

	select {
	case t.trkr.closeTagCh <- t:
		return nil
	default:
		return fmt.Errorf("无法关闭衰减标签 %s；队列已满（长度=%d）", t.name, len(t.trkr.closeTagCh))
	}
}
