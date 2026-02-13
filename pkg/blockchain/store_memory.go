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

import "sync"

// MemoryStore 内存存储结构体
type MemoryStore struct {
	sync.Mutex
	block *Block
}

// Add 添加区块到内存存储
// 参数 b 为要添加的区块
func (m *MemoryStore) Add(b Block) {
	m.Lock()
	m.block = &b
	m.Unlock()
}

// Len 返回存储中的区块数量
func (m *MemoryStore) Len() int {
	m.Lock()
	defer m.Unlock()
	if m.block == nil {
		return 0
	}
	return m.block.Index
}

// Last 返回最后一个区块
func (m *MemoryStore) Last() Block {
	m.Lock()
	defer m.Unlock()
	return *m.block
}
