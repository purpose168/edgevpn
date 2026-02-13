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

package cmd

import (
	"context"
	"time"

	"github.com/mudler/edgevpn/pkg/node"
	"github.com/mudler/edgevpn/pkg/services"
	"github.com/urfave/cli/v2"
)

func DNS() *cli.Command {
	return &cli.Command{
		Name:        "dns",
		Usage:       "启动本地 DNS 服务器",
		Description: `启动一个本地 DNS 服务器，使用区块链来解析地址`,
		UsageText:   "edgevpn dns",
		Flags: append(CommonFlags,
			&cli.StringFlag{
				Name:    "listen",
				Usage:   "DNS 监听地址。留空则禁用 DNS 服务器",
				EnvVars: []string{"DNSADDRESS"},
				Value:   "",
			},
			&cli.BoolFlag{
				Name:    "dns-forwarder",
				Usage:   "启用 DNS 转发",
				EnvVars: []string{"DNSFORWARD"},
				Value:   true,
			},
			&cli.IntFlag{
				Name:    "dns-cache-size",
				Usage:   "DNS LRU 缓存大小",
				EnvVars: []string{"DNSCACHESIZE"},
				Value:   200,
			},
			&cli.StringSliceFlag{
				Name:    "dns-forward-server",
				Usage:   "DNS 转发服务器列表，例如：8.8.8.8:53, 192.168.1.1:53 ...",
				EnvVars: []string{"DNSFORWARDSERVER"},
				Value:   cli.NewStringSlice("8.8.8.8:53", "1.1.1.1:53"),
			},
		),
		Action: func(c *cli.Context) error {
			o, _, ll := cliToOpts(c)

			dns := c.String("listen")
			// 添加 DNS 服务器
			o = append(o,
				services.DNS(ll, dns,
					c.Bool("dns-forwarder"),
					c.StringSlice("dns-forward-server"),
					c.Int("dns-cache-size"),
				)...)

			e, err := node.New(o...)
			if err != nil {
				return err
			}

			displayStart(ll)
			go handleStopSignals()

			ctx := context.Background()
			// 启动节点到网络，使用我们的账本
			if err := e.Start(ctx); err != nil {
				return err
			}

			for {
				time.Sleep(1 * time.Second)
			}
		},
	}
}
