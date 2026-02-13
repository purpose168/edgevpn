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

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/metrics"
	"github.com/purpose168/edgevpn/api"
	"github.com/purpose168/edgevpn/pkg/node"
	"github.com/purpose168/edgevpn/pkg/services"
	"github.com/urfave/cli/v2"
)

func API() *cli.Command {
	return &cli.Command{
		Name:  "api",
		Usage: "启动 HTTP 服务器以显示网络信息",
		Description: `在本地启动监听，为网络提供 API。
提供一个简单的 UI 界面来显示网络数据。`,
		UsageText: "edgevpn api",
		Flags: append(CommonFlags,
			&cli.BoolFlag{
				Name:    "enable-healthchecks",
				EnvVars: []string{"ENABLE_HEALTHCHECKS"},
			},
			&cli.BoolFlag{
				Name: "debug",
			},
			&cli.StringFlag{
				Name:  "listen",
				Value: "127.0.0.1:8080",
				Usage: "监听地址。要监听套接字，请使用 unix:// 前缀，例如 unix:///socket.path",
			},
		),
		Action: func(c *cli.Context) error {
			o, _, ll := cliToOpts(c)

			bwc := metrics.NewBandwidthCounter()
			o = append(o, node.WithLibp2pAdditionalOptions(libp2p.BandwidthReporter(bwc)))
			if c.Bool("enable-healthchecks") {
				o = append(o,
					services.Alive(
						time.Duration(c.Int("aliveness-healthcheck-interval"))*time.Second,
						time.Duration(c.Int("aliveness-healthcheck-scrub-interval"))*time.Second,
						time.Duration(c.Int("aliveness-healthcheck-max-interval"))*time.Second)...)
			}

			e, err := node.New(o...)
			if err != nil {
				return err
			}

			displayStart(ll)

			ctx := context.Background()
			go handleStopSignals()

			// 启动节点到网络，使用我们的账本
			if err := e.Start(ctx); err != nil {
				return err
			}

			return api.API(ctx, c.String("listen"), 5*time.Second, 20*time.Second, e, bwc, c.Bool("debug"))
		},
	}
}
