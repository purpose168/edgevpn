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
	"fmt"
	"os"
	"testing"
	"time"

	. "github.com/mudler/edgevpn/api/client"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// testInstance 从环境变量获取测试实例地址
var testInstance = os.Getenv("TEST_INSTANCE")

// TestClient 是客户端测试套件的入口函数
func TestClient(t *testing.T) {
	if testInstance == "" {
		fmt.Println("必须通过 TEST_INSTANCE 定义测试实例")
		os.Exit(1)
	}
	RegisterFailHandler(Fail)
	RunSpecs(t, "客户端测试套件")
}

var _ = BeforeSuite(func() {
	// 只有在有机器连接时才启动测试套件

	Eventually(func() (int, error) {
		c := NewClient(WithHost(testInstance))
		m, err := c.Machines()
		return len(m), err
	}, 100*time.Second, 1*time.Second).Should(BeNumerically(">=", 0))
})
