//go:build !windows && !darwin && !freebsd
// +build !windows,!darwin,!freebsd

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
	"github.com/mudler/water"
	"github.com/vishvananda/netlink"
)

// createInterface 在Linux等其他平台上创建网络接口
// 参数 c 为VPN配置
func createInterface(c *Config) (*water.Interface, error) {
	config := water.Config{
		DeviceType:             c.DeviceType,
		PlatformSpecificParams: water.PlatformSpecificParams{Persist: !c.NetLinkBootstrap},
	}
	config.Name = c.InterfaceName

	return water.New(config)
}

// prepareInterface 准备Linux等其他平台上的网络接口
// 使用netlink库配置接口的MTU、IP地址并启用接口
func prepareInterface(c *Config) error {
	// 根据名称获取网络链接
	link, err := netlink.LinkByName(c.InterfaceName)
	if err != nil {
		return err
	}

	// 解析IP地址
	addr, err := netlink.ParseAddr(c.InterfaceAddress)
	if err != nil {
		return err
	}

	// 设置MTU
	err = netlink.LinkSetMTU(link, c.InterfaceMTU)
	if err != nil {
		return err
	}

	// 添加IP地址
	err = netlink.AddrAdd(link, addr)
	if err != nil {
		return err
	}

	// 启用接口
	err = netlink.LinkSetUp(link)
	if err != nil {
		return err
	}
	return nil
}
