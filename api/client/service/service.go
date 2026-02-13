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
	"fmt"
	"strings"
	"time"

	edgeVPNClient "github.com/purpose168/edgevpn/api/client"
	"github.com/purpose168/edgevpn/pkg/protocol"
)

// Client 是 edgeVPN 客户端的封装
// 包含额外的元数据和语法糖
type Client struct {
	serviceID string
	*edgeVPNClient.Client
}

// NewClient 返回一个与指定服务 ID 关联的新客户端
func NewClient(serviceID string, c *edgeVPNClient.Client) *Client {
	return &Client{serviceID: serviceID, Client: c}
}

// ListItems 返回与 serviceID 和给定后缀关联的项目列表
func (c Client) ListItems(serviceID, suffix string) (strs []string, err error) {
	buckets, err := c.Client.GetBucketKeys(serviceID)
	if err != nil {
		return
	}
	for _, b := range buckets {
		if strings.HasSuffix(b, suffix) {
			b = strings.ReplaceAll(b, "-"+suffix, "")
			strs = append(strs, b)
		}
	}
	return
}

// advertizeMessage 广播消息结构体
type advertizeMessage struct {
	Time time.Time
}

// Advertize 将给定的 UUID 广播到账本
func (c Client) Advertize(uuid string) error {
	return c.Client.Put(c.serviceID, fmt.Sprintf("%s-uuid", uuid), advertizeMessage{Time: time.Now().UTC()})
}

// AdvertizingNodes 返回正在广播的节点列表
func (c Client) AdvertizingNodes() (active []string, err error) {
	uuids, err := c.ListItems(c.serviceID, "uuid")
	if err != nil {
		return
	}
	for _, u := range uuids {
		var d advertizeMessage
		res, err := c.Client.GetBucketKey(c.serviceID, fmt.Sprintf("%s-uuid", u))
		if err != nil {
			continue
		}
		res.Unmarshal(&d)

		// 检查是否在 15 分钟内活跃
		if d.Time.Add(15 * time.Minute).After(time.Now().UTC()) {
			active = append(active, u)
		}
	}
	return
}

// ActiveNodes 返回活跃节点列表
func (c Client) ActiveNodes() (active []string, err error) {
	res, err := c.Client.GetBucket(protocol.HealthCheckKey)
	if err != nil {
		return []string{}, err
	}

	for u, r := range res {
		var s string
		r.Unmarshal(&s)
		parsed, _ := time.Parse(time.RFC3339, s)
		// 检查是否在 15 分钟内活跃
		if parsed.Add(15 * time.Minute).After(time.Now().UTC()) {
			active = append(active, u)
		}
	}
	return
}

// Clean 清理与 serviceID 关联的数据
func (c Client) Clean() error {
	return c.Client.DeleteBucket(c.serviceID)
}

// reverse 反转字符串切片
func reverse(ss []string) {
	last := len(ss) - 1
	for i := 0; i < len(ss)/2; i++ {
		ss[i], ss[last-i] = ss[last-i], ss[i]
	}
}

// Get 从 API 获取通用数据
// 例如：get("ip", uuid)
func (c Client) Get(args ...string) (string, error) {
	reverse(args)
	key := strings.Join(args, "-")
	var role string
	d, err := c.Client.GetBucketKey(c.serviceID, key)
	if err == nil {
		d.Unmarshal(&role)
	}
	return role, err
}

// Set 向 API 设置通用数据
// 例如：set("ip", uuid, "value")
func (c Client) Set(thing, uuid, value string) error {
	return c.Client.Put(c.serviceID, fmt.Sprintf("%s-%s", uuid, thing), value)
}
