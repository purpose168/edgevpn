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

package utils

import "hash/fnv"

// hash 计算字符串的32位哈希值
// 参数 s 为要计算哈希的字符串
// 返回32位无符号整数哈希值
func hash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

// Leader 从活跃节点列表中选出领导者
// 使用一致性哈希算法，选择哈希值最大的节点作为领导者
// 参数 actives 为活跃节点ID列表
// 返回被选中的领导者节点ID
func Leader(actives []string) string {
	// 首先获取可用节点
	leaderboard := map[string]uint32{}

	leader := actives[0]

	// 计算当前谁是领导者
	for _, a := range actives {
		leaderboard[a] = hash(a)
		if leaderboard[leader] < leaderboard[a] {
			leader = a
		}
	}
	return leader
}
