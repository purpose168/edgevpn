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

	"github.com/purpose168/edgevpn/pkg/node"
	"github.com/urfave/cli/v2"
)

func Start() *cli.Command {
	return &cli.Command{
		Name:  "start",
		Usage: "启动网络而不激活任何接口",
		Description: `通过 P2P 网络连接，而不建立 VPN。
适用于设置中继或跳转节点以改善网络连接性。`,
		UsageText: "edgevpn start",
		Flags:     CommonFlags,
		Action: func(c *cli.Context) error {
			o, _, ll := cliToOpts(c)
			e, err := node.New(o...)
			if err != nil {
				return err
			}

			displayStart(ll)
			go handleStopSignals()

			// 启动节点到网络，使用我们的账本
			if err := e.Start(context.Background()); err != nil {
				return err
			}

			ll.Info("加入 P2P 网络")
			<-context.Background().Done()
			return nil
		},
	}
}
