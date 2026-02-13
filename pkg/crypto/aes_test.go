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

package crypto_test

import (
	. "github.com/mudler/edgevpn/pkg/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/mudler/edgevpn/pkg/crypto"
)

var _ = Describe("加密工具", func() {
	Context("AES", func() {
		It("编码/解码", func() {
			key := RandStringRunes(32)
			message := "foo"
			k := [32]byte{}
			copy([]byte(key)[:], k[:32])

			// 加密消息
			encoded, err := AESEncrypt(message, &k)
			Expect(err).ToNot(HaveOccurred())
			Expect(encoded).ToNot(Equal(key))
			Expect(len(encoded)).To(Equal(62))

			// 再次加密
			encoded2, err := AESEncrypt(message, &k)
			Expect(err).ToNot(HaveOccurred())

			// 应该不同（因为使用了随机nonce）
			Expect(encoded2).ToNot(Equal(encoded))

			// 解密并检查
			decoded, err := AESDecrypt(encoded, &k)
			Expect(err).ToNot(HaveOccurred())
			Expect(decoded).To(Equal(message))

			decoded, err = AESDecrypt(encoded2, &k)
			Expect(err).ToNot(HaveOccurred())
			Expect(decoded).To(Equal(message))
		})
	})
})
