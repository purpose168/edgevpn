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

import "encoding/json"

// Data 数据类型，表示存储在区块链中的数据
type Data string

// Unmarshal 将结果解析到接口。用于检索用SetValue设置的数据
// 参数 i 为目标接口指针
func (d Data) Unmarshal(i interface{}) error {
	return json.Unmarshal([]byte(d), i)
}
