// The MIT License (MIT)

// Copyright (c) 2017 Whyrusleeping (MIT)
// Copyright (c) 2022 Ettore Di Giacinto (Apache v2)

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
	"context"
	"errors"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/libp2p/go-libp2p/core/connmgr"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"

	logging "github.com/ipfs/go-log/v2"
)

var log = logging.Logger("peermgr")

// Manager 是一个连接管理器，当连接数超过高水位线时会修剪连接。
// 新连接在被修剪之前有一个宽限期。修剪会按需自动运行，
// 只有当距离上次修剪的时间超过10秒时才会执行。
// 此外，可以通过此结构体的公共接口显式请求修剪（参见 TrimOpenConns）。
//
// 参见 NewConnManager 中的配置参数。
type Manager struct {
	*decayer

	cfg      *config
	segments segments

	plk       sync.RWMutex
	protected map[peer.ID]map[string]struct{}

	// 基于通道的信号量，确保同时只有一个修剪在进行
	trimMutex sync.Mutex
	connCount int32
	// 以原子方式访问。这模仿了sync.Once的实现。
	// 修改此结构体时请注意正确的对齐方式。
	trimCount uint64

	lastTrimMu sync.RWMutex
	lastTrim   time.Time

	refCount                sync.WaitGroup
	ctx                     context.Context
	cancel                  func()
	unregisterMemoryWatcher func()
}

// segment 分段结构体，用于存储对等节点信息
type segment struct {
	sync.Mutex
	peers map[peer.ID]*peerInfo
}

// segments 分段数组，用于高效查找
type segments [256]*segment

// get 根据对等节点ID获取对应的分段
func (ss *segments) get(p peer.ID) *segment {
	return ss[byte(p[len(p)-1])]
}

// countPeers 统计所有分段中的对等节点数量
func (ss *segments) countPeers() (count int) {
	for _, seg := range ss {
		seg.Lock()
		count += len(seg.peers)
		seg.Unlock()
	}
	return count
}

// tagInfoFor 获取或创建对等节点的标签信息
// 参数 p 为对等节点ID
func (s *segment) tagInfoFor(p peer.ID) *peerInfo {
	pi, ok := s.peers[p]
	if ok {
		return pi
	}
	// 创建一个临时对等节点来缓冲早期标签，在Connected通知到达之前
	pi = &peerInfo{
		id:        p,
		firstSeen: time.Now(), // 当第一个Connected通知到达时，此时间戳将被更新
		temp:      true,
		tags:      make(map[string]int),
		decaying:  make(map[*decayingTag]*connmgr.DecayingValue),
		conns:     make(map[network.Stream]time.Time),
	}
	s.peers[p] = pi
	return pi
}

// NewConnManager 使用提供的参数创建新的Manager：
// lo和hi是水位线，控制将维护的连接数量。
// 当对等节点数超过"高水位线"时，将修剪尽可能多的对等节点（并终止其连接），
// 直到剩余"低水位线"个对等节点。
// 参数 low 为低水位线，hi 为高水位线，opts 为可选配置
func NewConnManager(low, hi int, opts ...Option) (*Manager, error) {
	cfg := &config{
		highWater:     hi,
		lowWater:      low,
		gracePeriod:   10 * time.Second,
		silencePeriod: 10 * time.Second,
	}
	for _, o := range opts {
		if err := o(cfg); err != nil {
			return nil, err
		}
	}

	if cfg.decayer == nil {
		// 设置默认的衰减器配置
		cfg.decayer = (&DecayerCfg{}).WithDefaults()
	}

	cm := &Manager{
		cfg:       cfg,
		protected: make(map[peer.ID]map[string]struct{}, 16),
		segments: func() (ret segments) {
			for i := range ret {
				ret[i] = &segment{
					peers: make(map[peer.ID]*peerInfo),
				}
			}
			return ret
		}(),
	}
	cm.ctx, cm.cancel = context.WithCancel(context.Background())

	decay, _ := NewDecayer(cfg.decayer, cm)
	cm.decayer = decay

	cm.refCount.Add(1)
	go cm.background()
	return cm, nil
}

// Close 关闭连接管理器
func (cm *Manager) Close() error {
	cm.cancel()
	if cm.unregisterMemoryWatcher != nil {
		cm.unregisterMemoryWatcher()
	}
	if err := cm.decayer.Close(); err != nil {
		return err
	}
	cm.refCount.Wait()
	return nil
}

// Protect 保护指定的对等节点不被修剪
// 参数 id 为对等节点ID，tag 为保护标签
func (cm *Manager) Protect(id peer.ID, tag string) {
	cm.plk.Lock()
	defer cm.plk.Unlock()

	tags, ok := cm.protected[id]
	if !ok {
		tags = make(map[string]struct{}, 2)
		cm.protected[id] = tags
	}
	tags[tag] = struct{}{}
}

// Unprotect 取消对指定对等节点的保护
// 参数 id 为对等节点ID，tag 为保护标签
// 返回该对等节点是否仍受保护
func (cm *Manager) Unprotect(id peer.ID, tag string) (protected bool) {
	cm.plk.Lock()
	defer cm.plk.Unlock()

	tags, ok := cm.protected[id]
	if !ok {
		return false
	}
	if delete(tags, tag); len(tags) == 0 {
		delete(cm.protected, id)
		return false
	}
	return true
}

// IsProtected 检查指定对等节点是否受保护
// 参数 id 为对等节点ID，tag 为保护标签（空字符串表示检查任何标签）
// 返回是否受保护
func (cm *Manager) IsProtected(id peer.ID, tag string) (protected bool) {
	cm.plk.Lock()
	defer cm.plk.Unlock()

	tags, ok := cm.protected[id]
	if !ok {
		return false
	}

	if tag == "" {
		return true
	}

	_, protected = tags[tag]
	return protected
}

// peerInfo 存储给定对等节点的元数据
type peerInfo struct {
	id       peer.ID
	tags     map[string]int                          // 每个标签的值
	decaying map[*decayingTag]*connmgr.DecayingValue // 衰减标签

	value int  // 所有标签值的缓存总和
	temp  bool // 这是一个临时条目，保存早期标签，等待连接

	conns map[network.Stream]time.Time // 每个连接的开始时间

	firstSeen time.Time // 我们开始跟踪此对等节点的时间戳
}

// peerInfos 对等节点信息切片
type peerInfos []peerInfo

// SortByValue 按值排序对等节点信息
func (p peerInfos) SortByValue() {
	sort.Slice(p, func(i, j int) bool {
		left, right := p[i], p[j]
		// 临时对等节点优先被修剪
		if left.temp != right.temp {
			return left.temp
		}
		// 否则，按值比较
		return left.value < right.value
	})
}

// TrimOpenConns 关闭尽可能多的对等节点的连接，使对等节点数等于低水位线。
// 对等节点按其总值的升序排序，优先修剪得分最低的对等节点，
// 只要它们不在宽限期内。
//
// 此函数会阻塞直到修剪完成。如果正在进行修剪，
// 则不会启动新的修剪，而是等待该修剪完成后再返回。
func (cm *Manager) TrimOpenConns(_ context.Context) {
	// TODO: 错误返回值，以便我们可以清晰地发出中止信号，因为：
	// (a) 有另一个修剪正在进行，或 (b) 静默期生效。

	cm.doTrim()
}

// background 后台协程，定期检查是否需要修剪
func (cm *Manager) background() {
	defer cm.refCount.Done()

	interval := cm.cfg.gracePeriod / 2
	if cm.cfg.silencePeriod != 0 {
		interval = cm.cfg.silencePeriod
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if atomic.LoadInt32(&cm.connCount) < int32(cm.cfg.highWater) {
				// 低于高水位线，跳过
				continue
			}
		case <-cm.ctx.Done():
			return
		}
		cm.trim()
	}
}

// doTrim 执行修剪操作
func (cm *Manager) doTrim() {
	// 此逻辑模仿标准库中sync.Once的实现
	count := atomic.LoadUint64(&cm.trimCount)
	cm.trimMutex.Lock()
	defer cm.trimMutex.Unlock()
	if count == atomic.LoadUint64(&cm.trimCount) {
		cm.trim()
		cm.lastTrimMu.Lock()
		cm.lastTrim = time.Now()
		cm.lastTrimMu.Unlock()
		atomic.AddUint64(&cm.trimCount, 1)
	}
}

// trim 开始修剪，如果上次修剪发生在配置的静默期之前
func (cm *Manager) trim() {
	// 执行实际的修剪
	for _, c := range cm.getConnsToClose() {
		c.Close()
	}
}

// getConnsToClose 运行TrimOpenConns中描述的启发式算法，返回要关闭的连接
func (cm *Manager) getConnsToClose() []network.Stream {
	if cm.cfg.lowWater == 0 || cm.cfg.highWater == 0 {
		// 已禁用
		return nil
	}

	if int(atomic.LoadInt32(&cm.connCount)) <= cm.cfg.lowWater {
		log.Info("打开的连接数低于限制")
		return nil
	}

	candidates := make(peerInfos, 0, cm.segments.countPeers())
	var ncandidates int
	gracePeriodStart := time.Now().Add(-cm.cfg.gracePeriod)

	cm.plk.RLock()
	for _, s := range cm.segments {
		s.Lock()
		for id, inf := range s.peers {
			if _, ok := cm.protected[id]; ok {
				// 跳过受保护的对等节点
				continue
			}
			if inf.firstSeen.After(gracePeriodStart) {
				// 跳过宽限期内的对等节点
				continue
			}
			// 注意，我们在这里复制条目，
			// 但由于inf.conns是一个map，它仍将指向原始对象
			candidates = append(candidates, *inf)
			ncandidates += len(inf.conns)
		}
		s.Unlock()
	}
	cm.plk.RUnlock()

	if ncandidates < cm.cfg.lowWater {
		log.Info("打开的连接数超过限制，但太多处于宽限期内")
		// 我们有太多连接，但超出宽限期的连接少于低水位线
		//
		// 如果现在修剪，可能会杀死有用的连接
		return nil
	}

	// 根据值排序对等节点
	candidates.SortByValue()

	target := ncandidates - cm.cfg.lowWater

	// 稍微多分配一些，因为每个对等节点可能有多个连接
	selected := make([]network.Stream, 0, target+10)

	for _, inf := range candidates {
		if target <= 0 {
			break
		}

		// 锁定以防止来自连接/断开事件的并发修改
		s := cm.segments.get(inf.id)
		s.Lock()
		if len(inf.conns) == 0 && inf.temp {
			// 处理早期标签的临时条目 -- 此条目已过宽限期
			// 但仍没有连接，因此修剪它
			delete(s.peers, inf.id)
		} else {
			for c := range inf.conns {
				selected = append(selected, c)
			}
			target -= len(inf.conns)
		}
		s.Unlock()
	}

	return selected
}

// GetTagInfo 获取与给定对等节点关联的标签信息
// 如果p引用的是未知对等节点，则返回nil
func (cm *Manager) GetTagInfo(p peer.ID) *connmgr.TagInfo {
	s := cm.segments.get(p)
	s.Lock()
	defer s.Unlock()

	pi, ok := s.peers[p]
	if !ok {
		return nil
	}

	out := &connmgr.TagInfo{
		FirstSeen: pi.firstSeen,
		Value:     pi.value,
		Tags:      make(map[string]int),
		Conns:     make(map[string]time.Time),
	}

	for t, v := range pi.tags {
		out.Tags[t] = v
	}
	for t, v := range pi.decaying {
		out.Tags[t.name] = v.Value
	}
	for c, t := range pi.conns {
		out.Conns[c.ID()] = t
	}

	return out
}

// TagPeer 将字符串和整数与给定对等节点关联
func (cm *Manager) TagPeer(p peer.ID, tag string, val int) {
	s := cm.segments.get(p)
	s.Lock()
	defer s.Unlock()

	pi := s.tagInfoFor(p)

	// 更新对等节点的总值
	pi.value += val - pi.tags[tag]
	pi.tags[tag] = val
}

// UntagPeer 取消字符串和整数与给定对等节点的关联
func (cm *Manager) UntagPeer(p peer.ID, tag string) {
	s := cm.segments.get(p)
	s.Lock()
	defer s.Unlock()

	pi, ok := s.peers[p]
	if !ok {
		log.Info("尝试从未跟踪的对等节点移除标签: ", p)
		return
	}

	// 更新对等节点的总值
	pi.value -= pi.tags[tag]
	delete(pi.tags, tag)
}

// UpsertTag 插入/更新对等节点标签
func (cm *Manager) UpsertTag(p peer.ID, tag string, upsert func(int) int) {
	s := cm.segments.get(p)
	s.Lock()
	defer s.Unlock()

	pi := s.tagInfoFor(p)

	oldval := pi.tags[tag]
	newval := upsert(oldval)
	pi.value += newval - oldval
	pi.tags[tag] = newval
}

// CMInfo 保存Manager的配置以及状态数据
type CMInfo struct {
	// 低水位线，如NewConnManager中所述
	LowWater int

	// 高水位线，如NewConnManager中所述
	HighWater int

	// 上次触发修剪的时间戳
	LastTrim time.Time

	// 配置的宽限期，如NewConnManager中所述
	GracePeriod time.Duration

	// 当前连接数
	ConnCount int
}

// GetInfo 返回此连接管理器的配置和状态数据
func (cm *Manager) GetInfo() CMInfo {
	cm.lastTrimMu.RLock()
	lastTrim := cm.lastTrim
	cm.lastTrimMu.RUnlock()

	return CMInfo{
		HighWater:   cm.cfg.highWater,
		LowWater:    cm.cfg.lowWater,
		LastTrim:    lastTrim,
		GracePeriod: cm.cfg.gracePeriod,
		ConnCount:   int(atomic.LoadInt32(&cm.connCount)),
	}
}

// HasStream 检索指定对等节点ID的流（如果存在）
// 参数 n 为网络，pid 为对等节点ID
func (cm *Manager) HasStream(n network.Network, pid peer.ID) (network.Stream, error) {
	s := cm.segments.get(pid)
	s.Lock()
	defer s.Unlock()

	pinfo, ok := s.peers[pid]
	if !ok {
		return nil, errors.New("该对等节点ID没有可用的流")
	}

	for c := range pinfo.conns {
		pinfo.conns[c] = time.Now()
		return c, nil
	}

	return nil, errors.New("没有可用的流")
}

// Connected 由通知器调用，通知已建立新连接。
// 通知器更新Manager以开始跟踪该连接。
// 如果新连接数超过高水位线，可能会触发修剪。
func (cm *Manager) Connected(n network.Network, c network.Stream) {

	p := c.Conn().RemotePeer()
	s := cm.segments.get(p)
	s.Lock()
	defer s.Unlock()

	pinfo, ok := s.peers[p]
	if !ok {
		pinfo = &peerInfo{
			id:        p,
			firstSeen: time.Now(),
			tags:      make(map[string]int),
			decaying:  make(map[*decayingTag]*connmgr.DecayingValue),
			conns:     make(map[network.Stream]time.Time),
		}
		s.peers[p] = pinfo
	} else if pinfo.temp {
		// 我们为此对等节点创建了一个临时条目，用于在Connected通知到达之前缓冲早期标签：
		// 翻转临时标志，并将firstSeen时间戳更新为真实值
		pinfo.temp = false
		pinfo.firstSeen = time.Now()
	}

	// 为对等节点缓存一个流
	pinfo.conns = map[network.Stream]time.Time{c: time.Now()}

	atomic.AddInt32(&cm.connCount, 1)
}

// Disconnected 由通知器调用，通知现有连接已关闭或终止。
// 通知器相应地更新Manager以停止跟踪该连接，并执行清理工作。
func (cm *Manager) Disconnected(n network.Network, c network.Stream) {

	p := c.Conn().RemotePeer()
	s := cm.segments.get(p)
	s.Lock()
	defer s.Unlock()

	cinf, ok := s.peers[p]
	if !ok {
		log.Error("收到未跟踪的对等节点的断开连接通知: ", p)
		return
	}

	_, ok = cinf.conns[c]
	if !ok {
		log.Error("收到未跟踪的连接的断开连接通知: ", p)
		return
	}

	delete(cinf.conns, c)
	if len(cinf.conns) == 0 {
		delete(s.peers, p)
	}
	atomic.AddInt32(&cm.connCount, -1)
}
