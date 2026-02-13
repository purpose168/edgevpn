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

	"github.com/purpose168/edgevpn/pkg/node"
	"github.com/purpose168/edgevpn/pkg/services"
	"github.com/urfave/cli/v2"
)

func cliNamePath(c *cli.Context) (name, path string, err error) {
	name = c.Args().Get(0)
	path = c.Args().Get(1)
	if name == "" && c.String("name") == "" {
		err = errors.New("需要提供文件 UUID 作为第一个参数或使用 --name 选项")
		return
	}
	if path == "" && c.String("path") == "" {
		err = errors.New("需要提供文件 UUID 作为第一个参数或使用 --name 选项")
		return
	}
	if c.String("name") != "" {
		name = c.String("name")
	}
	if c.String("path") != "" {
		path = c.String("path")
	}
	return name, path, nil
}

func FileSend() *cli.Command {
	return &cli.Command{
		Name:        "file-send",
		Aliases:     []string{"fs"},
		Usage:       "向网络提供文件服务",
		Description: `向网络提供文件服务，无需通过 VPN 连接`,
		UsageText:   "edgevpn file-send unique-id /src/path",
		Flags: append(CommonFlags,
			&cli.StringFlag{
				Name:     "name",
				Required: true,
				Usage: `在网络上提供服务的文件的唯一名称。
这也是接收文件时引用的 ID。`,
			},
			&cli.StringFlag{
				Name:     "path",
				Usage:    `要提供的文件`,
				Required: true,
			},
		),
		Action: func(c *cli.Context) error {
			name, path, err := cliNamePath(c)
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

			opts, err := services.ShareFile(ll, time.Duration(c.Int("ledger-announce-interval"))*time.Second, name, path)
			if err != nil {
				return err
			}
			o = append(o, opts...)

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

			for {
				time.Sleep(2 * time.Second)
			}
		},
	}
}

func FileReceive() *cli.Command {
	return &cli.Command{
		Name:        "file-receive",
		Aliases:     []string{"fr"},
		Usage:       "接收网络中提供的文件",
		Description: `从网络接收文件，无需通过 VPN 连接`,
		UsageText:   "edgevpn file-receive unique-id /dst/path",
		Flags: append(CommonFlags,
			&cli.StringFlag{
				Name:  "name",
				Usage: `要从网络接收的文件的唯一名称。`,
			},
			&cli.StringFlag{
				Name:  "path",
				Usage: `保存文件的目标位置`,
			},
		),
		Action: func(c *cli.Context) error {
			name, path, err := cliNamePath(c)
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

			ledger, _ := e.Ledger()

			return services.ReceiveFile(context.Background(), ledger, e, ll, time.Duration(c.Int("ledger-announce-interval"))*time.Second, name, path)
		},
	}
}
