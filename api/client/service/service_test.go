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

package service_test

import (
	"time"

	client "github.com/mudler/edgevpn/api/client"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/mudler/edgevpn/api/client/service"
)

var _ = Describe("Service", func() {
	c := client.NewClient(client.WithHost(testInstance))
	s := NewClient("foo", c)
	Context("检索节点", func() {
		PIt("检测节点", func() {
			Eventually(func() []string {
				n, _ := s.ActiveNodes()
				return n
			},
				100*time.Second, 1*time.Second).ShouldNot(BeEmpty())
		})
	})

	Context("广播节点", func() {
		It("检测节点", func() {
			n, err := s.AdvertizingNodes()
			Expect(len(n)).To(Equal(0))
			Expect(err).ToNot(HaveOccurred())

			s.Advertize("foo")

			Eventually(func() []string {
				n, _ := s.AdvertizingNodes()
				return n
			},
				100*time.Second, 1*time.Second).Should(Equal([]string{"foo"}))
		})
	})
})
