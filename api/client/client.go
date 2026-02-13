/*
Copyright © 2021-2022 Ettore Di Giacinto <mudler@mocaccino.org>
根据 Apache 许可证 2.0 版本（"许可证"）授权；
除非遵守许可证，否则您不得使用此文件。
您可以在以下位置获取许可证副本：
    http://www.apache.org/licenses/LICENSE-2.0
除非适用法律要求或书面同意，否则根据许可证分发的软件
是按"原样"分发的，没有任何明示或暗示的担保或条件。
请参阅许可证以了解管理权限和
限制的具体语言。
*/

package client

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/purpose168/edgevpn/api"
	"github.com/purpose168/edgevpn/pkg/blockchain"
	"github.com/purpose168/edgevpn/pkg/types"
)

type (
	// Client 结构体表示 API 客户端
	Client struct {
		host       string       // 主机地址
		httpClient *http.Client // HTTP 客户端
	}
)

// WithHost 返回一个配置客户端主机地址的选项函数
// 支持普通 HTTP 地址和 Unix 套接字地址（unix://）
func WithHost(host string) func(c *Client) error {
	return func(c *Client) error {
		c.host = host
		// 如果是 Unix 套接字地址
		if strings.HasPrefix(host, "unix://") {
			socket := strings.ReplaceAll(host, "unix://", "")
			c.host = "http://unix"
			c.httpClient = &http.Client{
				Transport: &http.Transport{
					DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
						return net.Dial("unix", socket)
					},
				},
			}
		}
		return nil
	}
}

// WithTimeout 返回一个配置客户端超时时间的选项函数
func WithTimeout(d time.Duration) func(c *Client) error {
	return func(c *Client) error {
		c.httpClient.Timeout = d
		return nil
	}
}

// WithHTTPClient 返回一个配置自定义 HTTP 客户端的选项函数
func WithHTTPClient(cl *http.Client) func(c *Client) error {
	return func(c *Client) error {
		c.httpClient = cl
		return nil
	}
}

// Option 定义客户端选项函数类型
type Option func(c *Client) error

// NewClient 创建一个新的 API 客户端实例
// 接受可变数量的选项函数来配置客户端
func NewClient(o ...Option) *Client {
	c := &Client{
		httpClient: &http.Client{},
	}
	// 应用所有配置选项
	for _, oo := range o {
		oo(c)
	}
	return c
}

// do 执行 HTTP 请求
// method: HTTP 方法（GET、POST、PUT、DELETE 等）
// endpoint: API 端点路径
// params: 查询参数映射
func (c *Client) do(method, endpoint string, params map[string]string) (*http.Response, error) {
	baseURL := fmt.Sprintf("%s%s", c.host, endpoint)

	req, err := http.NewRequest(method, baseURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	// 添加查询参数
	q := req.URL.Query()
	for key, val := range params {
		q.Set(key, val)
	}
	req.URL.RawQuery = q.Encode()
	return c.httpClient.Do(req)
}

// Get 方法（服务、用户、文件、账本、区块链、机器）

// Services 获取所有服务列表
func (c *Client) Services() (resp []types.Service, err error) {
	res, err := c.do(http.MethodGet, api.ServiceURL, nil)
	if err != nil {
		return
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return resp, err
	}
	if err = json.Unmarshal(body, &resp); err != nil {
		return resp, err
	}
	return
}

// Files 获取所有文件列表
func (c *Client) Files() (data []types.File, err error) {
	res, err := c.do(http.MethodGet, api.FileURL, nil)
	if err != nil {
		return
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return data, err
	}
	if err = json.Unmarshal(body, &data); err != nil {
		return data, err
	}
	return
}

// Users 获取所有用户列表
func (c *Client) Users() (data []types.User, err error) {
	res, err := c.do(http.MethodGet, api.UsersURL, nil)
	if err != nil {
		return
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return data, err
	}
	if err = json.Unmarshal(body, &data); err != nil {
		return data, err
	}
	return
}

// Ledger 获取完整账本数据
func (c *Client) Ledger() (data map[string]map[string]blockchain.Data, err error) {
	res, err := c.do(http.MethodGet, api.LedgerURL, nil)
	if err != nil {
		return
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return data, err
	}
	if err = json.Unmarshal(body, &data); err != nil {
		return data, err
	}
	return
}

// Summary 获取系统摘要信息
func (c *Client) Summary() (data types.Summary, err error) {
	res, err := c.do(http.MethodGet, api.SummaryURL, nil)
	if err != nil {
		return
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return data, err
	}
	if err = json.Unmarshal(body, &data); err != nil {
		return data, err
	}
	return
}

// Blockchain 获取区块链数据
func (c *Client) Blockchain() (data blockchain.Block, err error) {
	res, err := c.do(http.MethodGet, api.BlockchainURL, nil)
	if err != nil {
		return
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return data, err
	}
	if err = json.Unmarshal(body, &data); err != nil {
		return data, err
	}
	return
}

// Machines 获取所有机器列表
func (c *Client) Machines() (resp []types.Machine, err error) {
	res, err := c.do(http.MethodGet, api.MachineURL, nil)
	if err != nil {
		return
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return resp, err
	}
	if err = json.Unmarshal(body, &resp); err != nil {
		return resp, err
	}
	return
}

// GetBucket 获取指定存储桶的所有数据
// b: 存储桶名称
func (c *Client) GetBucket(b string) (resp map[string]blockchain.Data, err error) {
	res, err := c.do(http.MethodGet, fmt.Sprintf("%s/%s", api.LedgerURL, b), nil)
	if err != nil {
		return
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return resp, err
	}
	if err = json.Unmarshal(body, &resp); err != nil {
		return resp, err
	}
	return
}

// GetBucketKeys 获取指定存储桶中的所有键
// b: 存储桶名称
func (c *Client) GetBucketKeys(b string) (resp []string, err error) {
	d, err := c.GetBucket(b)
	if err != nil {
		return resp, err
	}
	// 提取所有键
	for k := range d {
		resp = append(resp, k)
	}
	return
}

// GetBuckets 获取所有存储桶名称列表
func (c *Client) GetBuckets() (resp []string, err error) {
	d, err := c.Ledger()
	if err != nil {
		return resp, err
	}
	// 提取所有存储桶名称
	for k := range d {
		resp = append(resp, k)
	}
	return
}

// GetBucketKey 获取指定存储桶中指定键的值
// b: 存储桶名称
// k: 键名
func (c *Client) GetBucketKey(b, k string) (resp blockchain.Data, err error) {
	res, err := c.do(http.MethodGet, fmt.Sprintf("%s/%s/%s", api.LedgerURL, b, k), nil)
	if err != nil {
		return
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return resp, err
	}

	var r string
	if err = json.Unmarshal(body, &r); err != nil {
		return resp, err
	}

	if err = json.Unmarshal([]byte(r), &r); err != nil {
		return resp, err
	}

	// 解码 Base64 URL 编码的数据
	d, err := base64.URLEncoding.DecodeString(r)
	if err != nil {
		return resp, err
	}
	resp = blockchain.Data(string(d))
	return
}

// Put 向指定存储桶的指定键写入数据
// b: 存储桶名称
// k: 键名
// v: 要写入的值（任意类型）
func (c *Client) Put(b, k string, v interface{}) (err error) {
	s := struct{ State string }{}

	// 将值序列化为 JSON
	dat, err := json.Marshal(v)
	if err != nil {
		return
	}

	// 进行 Base64 URL 编码
	d := base64.URLEncoding.EncodeToString(dat)

	res, err := c.do(http.MethodPut, fmt.Sprintf("%s/%s/%s/%s", api.LedgerURL, b, k, d), nil)
	if err != nil {
		return
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if err = json.Unmarshal(body, &s); err != nil {
		return err
	}

	// 检查返回状态是否为 "Announcing"
	if s.State != "Announcing" {
		return fmt.Errorf("意外的状态 '%s'", s.State)
	}

	return
}

// Delete 删除指定存储桶中的指定键
// b: 存储桶名称
// k: 键名
func (c *Client) Delete(b, k string) (err error) {
	s := struct{ State string }{}
	res, err := c.do(http.MethodDelete, fmt.Sprintf("%s/%s/%s", api.LedgerURL, b, k), nil)
	if err != nil {
		return
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(body, &s); err != nil {
		return err
	}
	// 检查返回状态是否为 "Announcing"
	if s.State != "Announcing" {
		return fmt.Errorf("意外的状态 '%s'", s.State)
	}

	return
}

// DeleteBucket 删除整个存储桶
// b: 存储桶名称
func (c *Client) DeleteBucket(b string) (err error) {
	s := struct{ State string }{}
	res, err := c.do(http.MethodDelete, fmt.Sprintf("%s/%s", api.LedgerURL, b), nil)
	if err != nil {
		return
	}
	defer res.Body.Close()
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if err = json.Unmarshal(body, &s); err != nil {
		return err
	}
	// 检查返回状态是否为 "Announcing"
	if s.State != "Announcing" {
		return fmt.Errorf("意外的状态 '%s'", s.State)
	}

	return
}
