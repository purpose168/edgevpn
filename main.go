/*
版权所有 © 2021-2022 Ettore Di Giacinto <mudler@mocaccino.org>
根据 Apache License 2.0（许可证）授权；
除非遵守许可证，否则您不得使用此文件。
您可以在以下地址获取许可证副本：
    http://www.apache.org/licenses/LICENSE-2.0
除非适用法律要求或书面同意，否则根据许可证分发的软件
按“原样”分发，不附带任何明示或暗示的担保或条件。
请参阅许可证以获取有关权限和
限制的具体语言。
*/

package main

// 生成 API 相关文件
//go:generate go run ./api/generate ./api/public/functions.tmpl ./api/public/index.tmpl ./api/public/index.html
import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/purpose168/edgevpn/cmd"
	internal "github.com/purpose168/edgevpn/internal"
)

// main 函数是应用程序的入口点
func main() {

	// 创建 CLI 应用程序实例
	app := &cli.App{
		Name:    "edgevpn",        // 应用名称
		Version: internal.Version, // 应用版本
		//Authors:     []*cli.Author{{Name: "Ettore Di Giacinto"}},  // 作者信息
		Usage:       "edgevpn --config /etc/edgevpn/config.yaml",  // 使用示例
		Description: "edgevpn 使用 libp2p 构建一个不可变的可信任区块链可寻址 P2P 网络", // 应用描述
		Copyright:   cmd.Copyright,                                // 版权信息
		Flags:       cmd.MainFlags(),                              // 主命令行标志
		Commands: []*cli.Command{ // 子命令列表
			cmd.Start(),          // 启动命令
			cmd.API(),            // API 命令
			cmd.ServiceAdd(),     // 添加服务命令
			cmd.ServiceConnect(), // 连接服务命令
			cmd.FileReceive(),    // 文件接收命令
			cmd.Proxy(),          // 代理命令
			cmd.FileSend(),       // 文件发送命令
			cmd.DNS(),            // DNS 命令
			cmd.Peergate(),       // 对等网关命令
		},

		Action: cmd.Main(), // 默认动作
	}

	// 运行应用程序
	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err) // 打印错误信息
		os.Exit(1)       // 以错误状态退出
	}
}
