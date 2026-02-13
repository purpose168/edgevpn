//go:build darwin
// +build darwin

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

package vpn

import (
	"net"
	"os/exec"
	"strconv"

	"github.com/mudler/water"
)

// createInterface 在macOS平台上创建网络接口
// 参数 c 为VPN配置
func createInterface(c *Config) (*water.Interface, error) {
	config := water.Config{
		DeviceType: water.TUN,
	}
	config.Name = c.InterfaceName

	return water.New(config)
}

// prepareInterface 准备macOS平台上的网络接口
// 使用ifconfig命令配置接口、MTU、IP地址和路由
func prepareInterface(c *Config) error {
	// 根据名称获取网络接口
	iface, err := net.InterfaceByName(c.InterfaceName)
	if err != nil {
		return err
	}

	// 解析CIDR格式的IP地址
	ip, ipNet, err := net.ParseCIDR(c.InterfaceAddress)
	if err != nil {
		return err
	}

	// 使用ifconfig命令设置MTU，因为net包不提供设置MTU的方法
	mtu := strconv.Itoa(c.InterfaceMTU)
	cmd := exec.Command("ifconfig", iface.Name, "mtu", mtu)
	err = cmd.Run()
	if err != nil {
		return err
	}

	// 将地址添加到接口。net包无法直接实现此功能，
	// 因此我们使用ifconfig命令。
	if ip.To4() == nil {
		// IPv6地址配置
		cmd = exec.Command("ifconfig", iface.Name, "inet6", ip.String())
	} else {
		// IPv4地址配置
		cmd = exec.Command("ifconfig", iface.Name, "inet", ip.String(), ip.String())
	}
	err = cmd.Run()
	if err != nil {
		return err
	}

	// 启用接口。net包无法直接实现此功能，
	// 因此我们使用ifconfig命令。
	cmd = exec.Command("ifconfig", iface.Name, "up")
	err = cmd.Run()
	if err != nil {
		return err
	}

	// 添加路由
	cmd = exec.Command("route", "-n", "add", "-net", ipNet.String(), ip.String())
	err = cmd.Run()
	if err != nil {
		return err
	}

	return nil
}
