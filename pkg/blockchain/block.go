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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// DataString 数据字符串类型
type DataString string

// Block 表示区块链中的每个"项目"
type Block struct {
	Index     int                        // 区块索引
	Timestamp string                     // 时间戳
	Storage   map[string]map[string]Data // 存储数据
	Hash      string                     // 当前区块哈希
	PrevHash  string                     // 前一区块哈希
}

// Blockchain 是一系列已验证的区块
type Blockchain []Block

// IsValid 通过检查索引和比较前一区块的哈希来确保区块有效
// 参数 oldBlock 为前一区块
func (newBlock Block) IsValid(oldBlock Block) bool {
	if oldBlock.Index+1 != newBlock.Index {
		return false
	}

	if oldBlock.Hash != newBlock.PrevHash {
		return false
	}

	if newBlock.Checksum() != newBlock.Hash {
		return false
	}

	return true
}

// Checksum 对区块进行SHA256哈希计算
func (b Block) Checksum() string {
	record := fmt.Sprint(b.Index, b.Timestamp, b.Storage, b.PrevHash)
	h := sha256.New()
	h.Write([]byte(record))
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed)
}

// NewBlock 使用前一区块的哈希创建新区块
// 参数 s 为存储数据
func (oldBlock Block) NewBlock(s map[string]map[string]Data) Block {
	var newBlock Block

	t := time.Now().UTC()

	newBlock.Index = oldBlock.Index + 1
	newBlock.Timestamp = t.String()
	newBlock.Storage = s
	newBlock.PrevHash = oldBlock.Hash
	newBlock.Hash = newBlock.Checksum()

	return newBlock
}
