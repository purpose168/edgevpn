// Copyright © 2021-2022 Ettore Di Giacinto <mudler@mocaccino.org>
//
// 本程序是自由软件；您可以根据自由软件基金会发布的
// GNU 通用公共许可证条款重新分发和/或修改它；
// 许可证版本 2 或（根据您的选择）任何后续版本。
//
// 分发本程序是希望它有用，
// 但没有任何保证；甚至没有适销性或特定用途适用性的
// 默示保证。请参阅
// GNU 通用公共许可证以获取更多详细信息。
//
// 您应该已经收到 GNU 通用公共许可证的副本
// 以及本程序；如果没有，请参阅 <http://www.gnu.org/licenses/>。

package service

import (
	"os/exec"
	"path/filepath"

	process "github.com/mudler/go-processmanager"
)

// NewProcessController 返回一个与状态目录关联的新进程控制器
func NewProcessController(statedir string) *ProcessController {
	return &ProcessController{stateDir: statedir}
}

// ProcessController go-processmanager 的语法糖封装
type ProcessController struct {
	stateDir string
}

// Process 返回一个与状态目录中二进制文件关联的进程
func (a *ProcessController) Process(state, p string, opts ...process.Option) *process.Process {
	return process.New(
		append(opts,
			process.WithName(a.BinaryPath(p)),
			process.WithStateDir(filepath.Join(a.stateDir, "proc", state)),
		)...,
	)
}

// BinaryPath 返回请求的程序二进制路径。
// 二进制路径相对于进程状态目录
func (a *ProcessController) BinaryPath(b string) string {
	return filepath.Join(a.stateDir, "bin", b)
}

// Run 从状态目录中的二进制文件运行命令
func (a *ProcessController) Run(command string, args ...string) (string, error) {
	cmd := exec.Command(a.BinaryPath(command), args...)
	out, err := cmd.CombinedOutput()

	return string(out), err
}
