//go:build windows
// +build windows

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
	"net/netip"

	"github.com/mudler/water"
	"golang.org/x/sys/windows"
	"golang.zx2c4.com/wireguard/windows/tunnel/winipcfg"
)

// prepareInterface 准备Windows平台上的网络接口
// 设置IP地址和MTU等网络参数
func prepareInterface(c *Config) error {
	// 查找由water创建的接口
	guid, err := windows.GUIDFromString("{00000000-FFFF-FFFF-FFE9-76E58C74063E}")
	if err != nil {
		return err
	}
	luid, err := winipcfg.LUIDFromGUID(&guid)
	if err != nil {
		return err
	}

	// 解析IP地址前缀
	prefix, err := netip.ParsePrefix(c.InterfaceAddress)
	if err != nil {
		return err
	}
	addresses := append([]netip.Prefix{}, prefix)
	// 设置接口IP地址
	if err := luid.SetIPAddresses(addresses); err != nil {
		return err
	}

	// 获取IPv4接口配置
	iface, err := luid.IPInterface(windows.AF_INET)
	if err != nil {
		return err
	}
	// 设置MTU值
	iface.NLMTU = uint32(c.InterfaceMTU)
	if err := iface.Set(); err != nil {
		return err
	}
	return nil
}

// createInterface 在Windows平台上创建网络接口
// 参数 c 为VPN配置
func createInterface(c *Config) (*water.Interface, error) {
	config := water.Config{
		DeviceType: c.DeviceType,
	}
	config.Name = c.InterfaceName
	return water.New(config)
}
