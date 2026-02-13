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
	"strings"

	"github.com/ipfs/go-log"
)

// Role 是服务角色。
// 它由一个唯一的字符串标识，通过网络发送
// 并流式传输到/从客户端。
// 角色可以直接应用，也可以在 API 中的角色内分配
type Role string

// RoleConfig 是角色配置结构，包含角色可以使用的所有对象
type RoleConfig struct {
	Client                                              *Client
	UUID, ServiceID, StateDir, APIAddress, NetworkToken string
	Logger                                              log.StandardLogger

	roles map[Role]func(c *RoleConfig) error
}

// RoleOption 是角色选项
type RoleOption func(c *RoleConfig)

// RoleKey 是角色（字符串）和处理程序之间的关联
// 处理程序实际完成角色功能
type RoleKey struct {
	RoleHandler func(c *RoleConfig) error
	Role        Role
}

// WithRole 设置可用角色
func WithRole(f map[Role]func(c *RoleConfig) error) RoleOption {
	return func(c *RoleConfig) {
		c.roles = f
	}
}

// WithRoleLogger 为角色操作设置日志记录器
func WithRoleLogger(l log.StandardLogger) RoleOption {
	return func(c *RoleConfig) {
		c.Logger = l
	}
}

// WithRoleUUID 设置执行角色的 UUID
func WithRoleUUID(u string) RoleOption {
	return func(c *RoleConfig) {
		c.UUID = u
	}
}

// WithRoleStateDir 设置角色的状态目录
func WithRoleStateDir(s string) RoleOption {
	return func(c *RoleConfig) {
		c.StateDir = s
	}
}

// WithRoleToken 设置角色可以使用的网络令牌
func WithRoleToken(s string) RoleOption {
	return func(c *RoleConfig) {
		c.NetworkToken = s
	}
}

// WithRoleAPIAddress 设置执行期间使用的 API 地址
func WithRoleAPIAddress(s string) RoleOption {
	return func(c *RoleConfig) {
		c.APIAddress = s
	}
}

// WithRoleServiceID 设置角色服务 ID
func WithRoleServiceID(s string) RoleOption {
	return func(c *RoleConfig) {
		c.ServiceID = s
	}
}

// WithRoleClient 为角色设置客户端
func WithRoleClient(e *Client) RoleOption {
	return func(c *RoleConfig) {
		c.Client = e
	}
}

// Apply 应用角色并接受选项列表
func (rr Role) Apply(opts ...RoleOption) {
	c := &RoleConfig{}
	for _, o := range opts {
		o(c)
	}

	for _, role := range strings.Split(string(rr), ",") {
		r := Role(role)
		if f, exists := c.roles[r]; exists {
			c.Logger.Info("角色已加载。正在应用 ", r)
			if err := f(c); err != nil {
				c.Logger.Warning("应用角色失败", role, err)
			}
		} else {
			c.Logger.Warn("未知角色: ", r)
		}
	}
}
