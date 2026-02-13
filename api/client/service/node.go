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
	"context"
	"embed"
	"encoding/base64"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/ipfs/go-log"
	"gopkg.in/yaml.v2"

	edgeVPNClient "github.com/mudler/edgevpn/api/client"
	edgevpn "github.com/mudler/edgevpn/pkg/node"
)

// Node 是服务节点。
// 它有一组定义的可用角色，网络中的节点可以承担这些角色。
// 它接受网络令牌或生成一个
type Node struct {
	stateDir                                 string  // 状态目录
	tokenFile                                string  // 令牌文件
	uuid                                     string  // 节点 UUID
	networkToken                             string  // 网络令牌
	apiAddress                               string  // API 地址
	defaultRoles, persistentRoles, stopRoles string  // 默认角色、持久角色、停止角色
	minNode                                  int     // 最小节点数

	assets []string      // 资产列表
	fs     embed.FS      // 嵌入的文件系统
	client *Client       // 客户端
	roles  map[Role]func(c *RoleConfig) error  // 角色映射

	logger log.StandardLogger  // 日志记录器
}

// WithRoles 定义一组角色键
func WithRoles(k ...RoleKey) Option {
	return func(mm *Node) error {
		m := map[Role]func(c *RoleConfig) error{}
		for _, kk := range k {
			m[kk.Role] = kk.RoleHandler
		}
		mm.roles = m
		return nil
	}
}

// WithMinNodes 设置最小节点数
func WithMinNodes(i int) Option {
	return func(k *Node) error {
		k.minNode = i
		return nil
	}
}

// WithFS 接受一个 embed.FS 文件系统，从中复制二进制文件
func WithFS(fs embed.FS) Option {
	return func(k *Node) error {
		k.fs = fs
		return nil
	}
}

// WithAssets 是要复制到临时状态目录的资产列表
// 它与 WithFS 一起使用以简化二进制嵌入
func WithAssets(assets ...string) Option {
	return func(k *Node) error {
		k.assets = assets
		return nil
	}
}

// WithLogger 定义在整个执行过程中使用的日志记录器
func WithLogger(l log.StandardLogger) Option {
	return func(k *Node) error {
		k.logger = l
		return nil
	}
}

// WithStopRoles 允许设置在清理期间应用的逗号分隔角色列表
func WithStopRoles(roles string) Option {
	return func(k *Node) error {
		k.stopRoles = roles
		return nil
	}
}

// WithPersistentRoles 允许设置持久应用的逗号分隔角色列表
func WithPersistentRoles(roles string) Option {
	return func(k *Node) error {
		k.persistentRoles = roles
		return nil
	}
}

// WithDefaultRoles 允许为节点设置逗号分隔的默认角色列表。
// 注意，设置此选项后，节点将拒绝任何分配的角色
func WithDefaultRoles(roles string) Option {
	return func(k *Node) error {
		k.defaultRoles = roles
		return nil
	}
}

// WithNetworkToken 允许设置网络令牌。
// 如果未设置，则会自动生成
func WithNetworkToken(token string) Option {
	return func(k *Node) error {
		k.networkToken = token
		return nil
	}
}

// WithAPIAddress 设置 EdgeVPN API 地址
func WithAPIAddress(s string) Option {
	return func(k *Node) error {
		k.apiAddress = s
		return nil
	}
}

// WithStateDir 设置节点状态目录。
// 它将包含解压的资产（如果有）和
// 角色生成的进程状态。
func WithStateDir(s string) Option {
	return func(k *Node) error {
		k.stateDir = s
		return nil
	}
}

// WithUUID 设置节点 UUID
func WithUUID(s string) Option {
	return func(k *Node) error {
		k.uuid = s
		return nil
	}
}

// WithTokenfile 设置令牌文件。
// 如果找不到令牌文件和网络令牌，则会写入该文件
func WithTokenfile(s string) Option {
	return func(k *Node) error {
		k.tokenFile = s
		return nil
	}
}

// WithClient 设置服务客户端
func WithClient(e *Client) Option {
	return func(o *Node) error {
		o.client = e
		return nil
	}
}

// Option 是节点选项
type Option func(k *Node) error

// NewNode 返回一个新的服务节点
// 服务节点可以应用角色，这些角色由 API 轮询。
// 这允许使用 API 协调节点来引导服务
// 并在之后应用角色（例如，使用动态接收的 IP 启动 VPN 等）
func NewNode(o ...Option) (*Node, error) {
	k := &Node{
		stateDir:   "/tmp/Node",
		apiAddress: "localhost:7070",
	}
	for _, oo := range o {
		err := oo(k)
		if err != nil {
			return nil, err
		}
	}
	return k, nil
}

// copyBinary 复制二进制文件到状态目录
func (k *Node) copyBinary() {
	for _, a := range k.assets {
		b := path.Base(a)
		aa := NewProcessController(k.stateDir)
		p := aa.BinaryPath(b)
		if _, err := os.Stat(p); err != nil {
			os.MkdirAll(filepath.Join(k.stateDir, "bin"), os.ModePerm)
			f, err := k.fs.Open(a)
			if err != nil {
				panic(err)
			}
			if err := copyFileContents(f, p); err != nil {
				panic(err)
			}
		}
	}
}

// copyFileContents 复制文件内容
func copyFileContents(in fs.File, dst string) (err error) {
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return
	}
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return
	}
	err = out.Sync()

	os.Chmod(dst, 0755)
	return
}

// Stop 通过调用停止角色来停止节点
func (k *Node) Stop() {
	k.execRoles(k.stopRoles)
}

// Clean 停止并清理节点
func (k *Node) Clean() {
	k.client.Clean()
	k.Stop()
	if k.stateDir != "" {
		os.RemoveAll(k.stateDir)
	}
}

// prepare 准备节点
func (k *Node) prepare() error {
	k.copyBinary()

	// 从令牌文件读取网络令牌
	if k.tokenFile != "" {
		f, err := ioutil.ReadFile(k.tokenFile)
		if err == nil {
			k.networkToken = string(f)
		}
	}

	// 如果没有网络令牌，生成一个新的
	if k.networkToken == "" {

		newData := edgevpn.GenerateNewConnectionData()
		bytesData, err := yaml.Marshal(newData)
		if err != nil {
			return err
		}

		token := base64.StdEncoding.EncodeToString(bytesData)

		k.logger.Infof("令牌已生成，写入到 '%s'", k.tokenFile)
		ioutil.WriteFile(k.tokenFile, []byte(token), os.ModePerm)
		k.networkToken = token
	}

	// 执行持久角色
	k.execRoles(k.persistentRoles)

	// 如果没有客户端，创建一个
	if k.client == nil {
		k.client = NewClient("Node",
			edgeVPNClient.NewClient(edgeVPNClient.WithHost(fmt.Sprintf("http://%s", k.apiAddress))))
	}
	return nil
}

// roleMessage 角色消息结构体
type roleMessage struct {
	Role Role
}

// options 返回角色选项列表
func (k *Node) options() (r []RoleOption) {
	r = []RoleOption{
		WithRoleLogger(k.logger),
		WithRole(k.roles),
		WithRoleClient(k.client),
		WithRoleUUID(k.uuid),
		WithRoleStateDir(k.stateDir),
		WithRoleAPIAddress(k.apiAddress),
		WithRoleToken(k.networkToken),
	}
	return
}

// execRoles 执行指定的角色
func (k *Node) execRoles(s string) {
	r := Role(s)
	k.logger.Infof("正在应用角色 '%s'", r)

	r.Apply(k.options()...)
}

// Start 使用上下文启动节点
func (k *Node) Start(ctx context.Context) error {
	// 准备二进制文件并启动默认角色
	if err := k.prepare(); err != nil {
		return err
	}

	k.client.Advertize(k.uuid)

	minNode := 2
	if k.minNode != 0 {
		minNode = k.minNode
	}

	i := 0
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			i++
			time.Sleep(10 * time.Second)
			// 每 20 秒广播一次
			if i%2 == 0 {
				k.client.Advertize(k.uuid)
			}

			uuids, _ := k.client.ActiveNodes()

			for _, n := range uuids {
				k.logger.Infof("活跃: '%s'", n)
			}

			// 如果有持久角色，执行它们
			if k.persistentRoles != "" {
				k.execRoles(k.persistentRoles)
			}

			// 如果有默认角色，执行它们并继续
			if k.defaultRoles != "" {
				k.execRoles(k.defaultRoles)
				continue
			}

			// 节点数不足
			if len(uuids) < minNode {
				k.logger.Infof("可用节点不足，正在休眠... 需要: %d, 可用: %d", minNode, len(uuids))
				continue
			}

			// 足够的活跃节点。
			d, err := k.client.Get("role", k.uuid)
			if err == nil {
				k.logger.Info("角色已分配")
				k.execRoles(d)
			} else {
				// 还没有角色，正在休眠
				k.logger.Info("未分配角色，正在休眠")
			}

		}
	}
}
