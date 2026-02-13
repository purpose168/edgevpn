/*
Copyright © 2022 Ettore Di Giacinto <mudler@mocaccino.org>
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

package crypto

import (
	"encoding/base64"
	"hash"

	"github.com/creachadair/otp"
)

// TOTP 生成基于时间的一次性密码
// 参数 f 为哈希函数，digits 为输出位数，t 为时间步长，key 为密钥
// 返回TOTP字符串
func TOTP(f func() hash.Hash, digits int, t int, key string) string {
	cfg := otp.Config{
		Hash:     f,      // 默认为sha1.New
		Digits:   digits, // 默认为6
		TimeStep: otp.TimeWindow(t),
		Key:      key,
		Format: func(hash []byte, nb int) string {
			return base64.StdEncoding.EncodeToString(hash)[:nb]
		},
	}
	return cfg.TOTP()
}
