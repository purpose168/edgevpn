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
	"errors"
	"time"

	"github.com/mudler/edgevpn/pkg/node"
	"github.com/mudler/edgevpn/pkg/services"
	"github.com/urfave/cli/v2"
)

func cliNameAddress(c *cli.Context) (name, address string, err error) {
	name = c.Args().Get(0)
	address = c.Args().Get(1)
	if name == "" && c.String("name") == "" {
		err = errors.New("需要提供文件 UUID 作为第一个参数或使用 --name 选项")
		return
	}
	if address == "" && c.String("address") == "" {
		err = errors.New("需要提供文件 UUID 作为第一个参数或使用 --name 选项")
		return
	}
	if c.String("name") != "" {
		name = c.String("name")
	}
	if c.String("address") != "" {
		address = c.String("address")
	}
	return name, address, nil
}

func ServiceAdd() *cli.Command {
	return &cli.Command{
		Name:    "service-add",
		Aliases: []string{"sa"},
		Usage:   "向网络暴露服务而不创建 VPN",
		Description: `将本地或远程端点连接作为 VPN 中的服务暴露。
		主机将充当服务与连接之间的代理`,
		UsageText: "edgevpn service-add unique-id ip:port",
		Flags: append(CommonFlags,
			&cli.StringFlag{
				Name:  "name",
				Usage: `在网络上提供服务的唯一名称。`,
			},
			&cli.StringFlag{
				Name: "address",
				Usage: `服务运行的远程地址。可以是远程 Web 服务器、本地 SSH 服务器等。
例如，'192.168.1.1:80' 或 '127.0.0.1:22'。`,
			},
		),
		Action: func(c *cli.Context) error {
			name, address, err := cliNameAddress(c)
			if err != nil {
				return err
			}
			o, _, ll := cliToOpts(c)

			// 需要解除低活动连接的阻塞
			o = append(o,
				services.Alive(
					time.Duration(c.Int("aliveness-healthcheck-interval"))*time.Second,
					time.Duration(c.Int("aliveness-healthcheck-scrub-interval"))*time.Second,
					time.Duration(c.Int("aliveness-healthcheck-max-interval"))*time.Second)...)

			o = append(o, services.RegisterService(ll, time.Duration(c.Int("ledger-announce-interval"))*time.Second, name, address)...)

			e, err := node.New(o...)
			if err != nil {
				return err
			}

			displayStart(ll)
			go handleStopSignals()

			// 将节点加入网络，使用我们的账本
			if err := e.Start(context.Background()); err != nil {
				return err
			}

			for {
				time.Sleep(2 * time.Second)
			}
		},
	}
}

func ServiceConnect() *cli.Command {
	return &cli.Command{
		Aliases: []string{"sc"},
		Usage:   "连接到网络中的服务而不创建 VPN",
		Name:    "service-connect",
		Description: `绑定本地端口以连接到网络中的远程服务。
创建一个本地监听器，通过网络连接到服务而不创建 VPN。
`,
		UsageText: "edgevpn service-connect unique-id (ip):port",
		Flags: append(CommonFlags,
			&cli.StringFlag{
				Name:  "name",
				Usage: `网络中服务的唯一名称。`,
			},
			&cli.StringFlag{
				Name: "address",
				Usage: `本地绑定的地址。例如 ':8080'。将创建一个代理
连接到网络中的服务`,
			},
		),
		Action: func(c *cli.Context) error {
			name, address, err := cliNameAddress(c)
			if err != nil {
				return err
			}
			o, _, ll := cliToOpts(c)

			// 需要解除低活动连接的阻塞
			o = append(o,
				services.Alive(
					time.Duration(c.Int("aliveness-healthcheck-interval"))*time.Second,
					time.Duration(c.Int("aliveness-healthcheck-scrub-interval"))*time.Second,
					time.Duration(c.Int("aliveness-healthcheck-max-interval"))*time.Second)...)

			e, err := node.New(
				append(o,
					node.WithNetworkService(
						services.ConnectNetworkService(
							time.Duration(c.Int("ledger-announce-interval"))*time.Second,
							name,
							address,
						),
					),
				)...,
			)
			if err != nil {
				return err
			}
			displayStart(ll)
			go handleStopSignals()

			// 启动节点
			return e.Start(context.Background())
		},
	}
}
