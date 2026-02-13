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

package blockchain

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"sync"
	"time"

	"github.com/mudler/edgevpn/pkg/hub"
	"github.com/mudler/edgevpn/pkg/utils"

	"github.com/pkg/errors"
)

// Ledger 账本结构体
type Ledger struct {
	sync.Mutex
	blockchain Store // 区块链存储

	channel io.Writer // 写入通道
}

// Store 存储接口
type Store interface {
	Add(Block)   // 添加区块
	Len() int    // 返回长度
	Last() Block // 返回最后区块
}

// New 创建新的账本，写入到指定的writer
// 参数 w 为写入器，s 为存储器
func New(w io.Writer, s Store) *Ledger {
	c := &Ledger{channel: w, blockchain: s}
	if s.Len() == 0 {
		c.newGenesis()
	}
	return c
}

// newGenesis 创建创世区块
func (l *Ledger) newGenesis() {
	t := time.Now()
	genesisBlock := Block{}
	genesisBlock = Block{0, t.String(), map[string]map[string]Data{}, genesisBlock.Checksum(), ""}
	l.blockchain.Add(genesisBlock)
}

// Syncronizer 启动一个goroutine，定期将区块链写入通道
// 参数 ctx 为上下文，t 为时间间隔
func (l *Ledger) Syncronizer(ctx context.Context, t time.Duration) {
	go func() {
		t := utils.NewBackoffTicker(utils.BackoffMaxInterval(t))
		defer t.Stop()
		for {
			select {
			case <-t.C:
				l.Lock()

				bytes, err := json.Marshal(l.blockchain.Last())
				if err != nil {
					log.Println(err)
				}

				l.channel.Write(compress(bytes).Bytes())

				l.Unlock()
			case <-ctx.Done():
				return
			}
		}
	}()
}

// compress 压缩字节数据
// 参数 b 为要压缩的字节数据
func compress(b []byte) *bytes.Buffer {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	gz.Write(b)
	gz.Close()
	return &buf
}

// deCompress 解压字节数据
// 参数 b 为要解压的字节数据
func deCompress(b []byte) (*bytes.Buffer, error) {
	r, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	result, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	return bytes.NewBuffer(result), nil
}

// Update 从消息更新区块链
// 参数 f 为账本，h 为消息，c 为消息通道
func (l *Ledger) Update(f *Ledger, h *hub.Message, c chan *hub.Message) (err error) {
	block := &Block{}

	b, err := deCompress([]byte(h.Message))
	if err != nil {
		err = errors.Wrap(err, "解压失败")
		return
	}

	err = json.Unmarshal(b.Bytes(), block)
	if err != nil {
		err = errors.Wrap(err, "解析区块链数据失败")
		return
	}

	l.Lock()
	if block.Index > l.blockchain.Len() {
		l.blockchain.Add(*block)
	}
	l.Unlock()

	return
}

// Announce 持续异步更新数据到区块链。
// 在指定间隔发送广播，
// 确保异步获取的值被写入区块链
// 参数 ctx 为上下文，d 为时间间隔，async 为异步函数
func (l *Ledger) Announce(ctx context.Context, d time.Duration, async func()) {
	go func() {
		t := utils.NewBackoffTicker(utils.BackoffMaxInterval(d))
		defer t.Stop()
		for {
			select {
			case <-t.C:
				async()

			case <-ctx.Done():
				return
			}
		}
	}()
}

// AnnounceDeleteBucket 公告删除存储桶。当存储桶被删除时停止
// 接受间隔时间和最大超时时间。
// 这是尽力而为的，超时是必要的，否则如果有多个写入者尝试写入同一资源，可能会淹没网络请求
// 参数 ctx 为上下文，interval 为间隔时间，timeout 为超时时间，bucket 为存储桶名称
func (l *Ledger) AnnounceDeleteBucket(ctx context.Context, interval, timeout time.Duration, bucket string) {
	del, cancel := context.WithTimeout(ctx, timeout)

	l.Announce(del, interval, func() {
		_, exists := l.CurrentData()[bucket]
		if exists {
			l.DeleteBucket(bucket)
		} else {
			cancel()
		}
	})
}

// AnnounceDeleteBucketKey 公告删除存储桶中的键。当键被删除时停止
// 参数 ctx 为上下文，interval 为间隔时间，timeout 为超时时间，bucket 为存储桶名称，key 为键名
func (l *Ledger) AnnounceDeleteBucketKey(ctx context.Context, interval, timeout time.Duration, bucket, key string) {
	del, cancel := context.WithTimeout(ctx, timeout)

	l.Announce(del, interval, func() {
		_, exists := l.CurrentData()[bucket][key]
		if exists {
			l.Delete(bucket, key)
		} else {
			cancel()
		}
	})
}

// AnnounceUpdate 如果状态不同，持续向区块链公告内容
// 参数 ctx 为上下文，interval 为间隔时间，bucket 为存储桶名称，key 为键名，value 为值
func (l *Ledger) AnnounceUpdate(ctx context.Context, interval time.Duration, bucket, key string, value interface{}) {
	l.Announce(ctx, interval, func() {
		v, exists := l.CurrentData()[bucket][key]
		realv, _ := json.Marshal(value)
		switch {
		case !exists || string(v) != string(realv):
			l.Add(bucket, map[string]interface{}{key: value})
		}
	})
}

// Persist 持续向区块链公告内容，直到协调完成
// 参数 ctx 为上下文，interval 为间隔时间，timeout 为超时时间，bucket 为存储桶名称，key 为键名，value 为值
func (l *Ledger) Persist(ctx context.Context, interval, timeout time.Duration, bucket, key string, value interface{}) {
	put, cancel := context.WithTimeout(ctx, timeout)

	l.Announce(put, interval, func() {
		v, exists := l.CurrentData()[bucket][key]
		realv, _ := json.Marshal(value)
		switch {
		case !exists || string(v) != string(realv):
			l.Add(bucket, map[string]interface{}{key: value})
		case exists && string(v) == string(realv):
			cancel()
		}
	})
}

// GetKey 从区块链检索当前键
// 参数 b 为存储桶名称，s 为键名
func (l *Ledger) GetKey(b, s string) (value Data, exists bool) {
	l.Lock()
	defer l.Unlock()

	if l.blockchain.Len() > 0 {
		last := l.blockchain.Last()
		if _, exists = last.Storage[b]; !exists {
			return
		}
		value, exists = last.Storage[b][s]
		if exists {
			return
		}
	}
	return
}

// Exists 如果存在一个匹配值的元素则返回true
// 参数 b 为存储桶名称，f 为匹配函数
func (l *Ledger) Exists(b string, f func(Data) bool) (exists bool) {
	l.Lock()
	defer l.Unlock()
	if l.blockchain.Len() > 0 {
		for _, bv := range l.blockchain.Last().Storage[b] {
			if f(bv) {
				exists = true
				return
			}
		}
	}

	return
}

// CurrentData 返回当前账本数据（加锁）
func (l *Ledger) CurrentData() map[string]map[string]Data {
	l.Lock()
	defer l.Unlock()

	return buckets(l.blockchain.Last().Storage).copy()
}

// LastBlock 返回区块链中的最后一个区块
func (l *Ledger) LastBlock() Block {
	l.Lock()
	defer l.Unlock()
	return l.blockchain.Last()
}

// bucket 存储桶类型
type bucket map[string]Data

// copy 复制存储桶
func (b bucket) copy() map[string]Data {
	copy := map[string]Data{}
	for k, v := range b {
		copy[k] = v
	}
	return copy
}

// buckets 存储桶集合类型
type buckets map[string]map[string]Data

// copy 复制存储桶集合
func (b buckets) copy() map[string]map[string]Data {
	copy := map[string]map[string]Data{}
	for k, v := range b {
		copy[k] = bucket(v).copy()
	}
	return copy
}

// Add 向区块链添加数据
// 参数 b 为存储桶名称，s 为键值对映射
func (l *Ledger) Add(b string, s map[string]interface{}) {
	l.Lock()
	current := buckets(l.blockchain.Last().Storage).copy()

	for s, k := range s {
		if _, exists := current[b]; !exists {
			current[b] = make(map[string]Data)
		}
		dat, _ := json.Marshal(k)
		current[b][s] = Data(string(dat))
	}
	l.Unlock()
	l.writeData(current)
}

// Delete 从账本删除数据（加锁）
// 参数 b 为存储桶名称，k 为键名
func (l *Ledger) Delete(b string, k string) {
	l.Lock()
	new := make(map[string]map[string]Data)
	for bb, kk := range l.blockchain.Last().Storage {
		if _, exists := new[bb]; !exists {
			new[bb] = make(map[string]Data)
		}
		// 复制除b/k以外的所有键值
		for kkk, v := range kk {
			if !(bb == b && kkk == k) {
				new[bb][kkk] = v
			}
		}
	}
	l.Unlock()
	l.writeData(new)
}

// DeleteBucket 从账本删除存储桶（加锁）
// 参数 b 为存储桶名称
func (l *Ledger) DeleteBucket(b string) {
	l.Lock()
	new := make(map[string]map[string]Data)
	for bb, kk := range l.blockchain.Last().Storage {
		// 复制除指定存储桶以外的所有内容
		if bb == b {
			continue
		}
		if _, exists := new[bb]; !exists {
			new[bb] = make(map[string]Data)
		}
		for kkk, v := range kk {
			new[bb][kkk] = v
		}
	}
	l.Unlock()
	l.writeData(new)
}

// String 返回区块链的字符串表示
func (l *Ledger) String() string {
	bytes, _ := json.MarshalIndent(l.blockchain, "", "  ")
	return string(bytes)
}

// Index 返回最后已知的区块链索引
func (l *Ledger) Index() int {
	return l.blockchain.Len()
}

// writeData 写入数据到区块链
// 参数 s 为数据映射
func (l *Ledger) writeData(s map[string]map[string]Data) {
	newBlock := l.blockchain.Last().NewBlock(s)

	if newBlock.IsValid(l.blockchain.Last()) {
		l.Lock()
		l.blockchain.Add(newBlock)
		l.Unlock()
	}

	bytes, err := json.Marshal(l.blockchain.Last())
	if err != nil {
		log.Println(err)
	}

	l.channel.Write(compress(bytes).Bytes())
}
