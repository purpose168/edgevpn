/*
Copyright © 2021-2022 Ettore Di Giacinto <mudler@mocaccino.org>
根据 Apache 许可证 2.0 版本（"许可证"）授权；
除非遵守许可证，否则您不得使用此文件。
您可以在以下位置获取许可证副本：
    http://www.apache.org/licenses/LICENSE-2.0
除非适用法律要求或书面同意，否则根据许可证分发的软件
是按"原样"分发的，没有任何明示或暗示的担保或条件。
请参阅许可证以了解管理权限和
限制的具体语言。
*/

package client_test

import (
	"math/rand"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/mudler/edgevpn/api/client"
)

// letterBytes 用于生成随机字符串的字母表
const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// randStringBytes 生成指定长度的随机字符串
func randStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

var _ = Describe("Client", func() {
	c := NewClient(WithHost(testInstance))

	Context("操作区块链", func() {
		var testBucket string

		AfterEach(func() {
			// 验证存储桶存在后删除
			Eventually(c.GetBuckets, 100*time.Second, 1*time.Second).Should(ContainElement(testBucket))
			err := c.DeleteBucket(testBucket)
			Expect(err).ToNot(HaveOccurred())
			// 验证存储桶已被删除
			Eventually(c.GetBuckets, 100*time.Second, 1*time.Second).ShouldNot(ContainElement(testBucket))
		})

		BeforeEach(func() {
			// 为每个测试生成随机存储桶名称
			testBucket = randStringBytes(10)
		})

		It("写入字符串数据", func() {
			err := c.Put(testBucket, "foo", "bar")
			Expect(err).ToNot(HaveOccurred())

			// 验证存储桶已创建
			Eventually(c.GetBuckets, 100*time.Second, 1*time.Second).Should(ContainElement(testBucket))
			// 验证键已创建
			Eventually(func() ([]string, error) { return c.GetBucketKeys(testBucket) }, 100*time.Second, 1*time.Second).Should(ContainElement("foo"))

			// 验证值可以正确读取
			Eventually(func() (string, error) {
				resp, err := c.GetBucketKey(testBucket, "foo")
				if err == nil {
					var r string
					resp.Unmarshal(&r)
					return r, nil
				}
				return "", err
			}, 100*time.Second, 1*time.Second).Should(Equal("bar"))

			// 验证账本数据
			m, err := c.Ledger()
			Expect(err).ToNot(HaveOccurred())
			Expect(len(m) > 0).To(BeTrue())
		})

		It("写入随机数据", func() {
			err := c.Put(testBucket, "foo2", struct{ Foo string }{Foo: "bar"})
			Expect(err).ToNot(HaveOccurred())
			// 验证结构体数据可以正确读取
			Eventually(func() (string, error) {
				resp, err := c.GetBucketKey(testBucket, "foo2")
				if err == nil {
					var r struct{ Foo string }
					resp.Unmarshal(&r)
					return r.Foo, nil
				}

				return "", err
			}, 100*time.Second, 1*time.Second).Should(Equal("bar"))
		})
	})
})
