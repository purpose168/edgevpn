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

package api

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	_ "net/http/pprof"
	"path/filepath"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/metrics"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	p2pprotocol "github.com/libp2p/go-libp2p/core/protocol"

	"github.com/miekg/dns"
	apiTypes "github.com/purpose168/edgevpn/api/types"

	"github.com/labstack/echo/v4"
	"github.com/purpose168/edgevpn/pkg/node"
	"github.com/purpose168/edgevpn/pkg/protocol"
	"github.com/purpose168/edgevpn/pkg/services"
	"github.com/purpose168/edgevpn/pkg/types"
)

//go:embed public
var embededFiles embed.FS

// getFileSystem 获取嵌入的文件系统
func getFileSystem() http.FileSystem {
	fsys, err := fs.Sub(embededFiles, "public")
	if err != nil {
		panic(err)
	}

	return http.FS(fsys)
}

// API 端点常量定义
const (
	MachineURL    = "/api/machines"   // 机器列表端点
	UsersURL      = "/api/users"      // 用户列表端点
	ServiceURL    = "/api/services"   // 服务列表端点
	BlockchainURL = "/api/blockchain" // 区块链端点
	LedgerURL     = "/api/ledger"     // 账本端点
	SummaryURL    = "/api/summary"    // 摘要端点
	FileURL       = "/api/files"      // 文件列表端点
	NodesURL      = "/api/nodes"      // 节点列表端点
	DNSURL        = "/api/dns"        // DNS 端点
	MetricsURL    = "/api/metrics"    // 指标端点
	PeerstoreURL  = "/api/peerstore"  // 对等存储端点
	PeerGateURL   = "/api/peergate"   // 对等网关端点
)

// API 启动 EdgeVPN API 服务器
// ctx: 上下文
// l: 监听地址（支持 unix:// 前缀的 Unix 套接字）
// defaultInterval: 默认间隔时间
// timeout: 超时时间
// e: EdgeVPN 节点实例
// bwc: 带宽报告器
// debugMode: 是否启用调试模式
func API(ctx context.Context, l string, defaultInterval, timeout time.Duration, e *node.Node, bwc metrics.Reporter, debugMode bool) error {

	ledger, _ := e.Ledger()

	ec := echo.New()

	// 支持 Unix 套接字监听
	if strings.HasPrefix(l, "unix://") {
		unixListener, err := net.Listen("unix", strings.ReplaceAll(l, "unix://", ""))
		if err != nil {
			return err
		}
		ec.Listener = unixListener
	}

	assetHandler := http.FileServer(getFileSystem())
	// 调试模式下启用 pprof
	if debugMode {
		ec.GET("/debug/pprof/*", echo.WrapHandler(http.DefaultServeMux))
	}

	// 带宽指标端点
	if bwc != nil {
		ec.GET(MetricsURL, func(c echo.Context) error {
			return c.JSON(http.StatusOK, bwc.GetBandwidthTotals())
		})
		ec.GET(filepath.Join(MetricsURL, "protocol"), func(c echo.Context) error {
			return c.JSON(http.StatusOK, bwc.GetBandwidthByProtocol())
		})
		ec.GET(filepath.Join(MetricsURL, "peer"), func(c echo.Context) error {
			return c.JSON(http.StatusOK, bwc.GetBandwidthByPeer())
		})
		ec.GET(filepath.Join(MetricsURL, "peer", ":peer"), func(c echo.Context) error {
			return c.JSON(http.StatusOK, bwc.GetBandwidthForPeer(peer.ID(c.Param("peer"))))
		})
		ec.GET(filepath.Join(MetricsURL, "protocol", ":protocol"), func(c echo.Context) error {
			return c.JSON(http.StatusOK, bwc.GetBandwidthForProtocol(p2pprotocol.ID(c.Param("protocol"))))
		})
	}
	// 从账本获取文件数据
	ec.GET(FileURL, func(c echo.Context) error {
		list := []*types.File{}
		for _, v := range ledger.CurrentData()[protocol.FilesLedgerKey] {
			machine := &types.File{}
			v.Unmarshal(machine)
			list = append(list, machine)
		}
		return c.JSON(http.StatusOK, list)
	})

	// 对等网关控制端点
	if e.PeerGater() != nil {
		ec.PUT(fmt.Sprintf("%s/:state", PeerGateURL), func(c echo.Context) error {
			state := c.Param("state")

			switch state {
			case "enable":
				e.PeerGater().Enable()
			case "disable":
				e.PeerGater().Disable()
			}
			return c.JSON(http.StatusOK, e.PeerGater().Enabled())
		})

		ec.GET(PeerGateURL, func(c echo.Context) error {
			return c.JSON(http.StatusOK, e.PeerGater().Enabled())
		})
	}

	// 系统摘要端点
	ec.GET(SummaryURL, func(c echo.Context) error {
		files := len(ledger.CurrentData()[protocol.FilesLedgerKey])
		machines := len(ledger.CurrentData()[protocol.MachinesLedgerKey])
		users := len(ledger.CurrentData()[protocol.UsersLedgerKey])
		services := len(ledger.CurrentData()[protocol.ServicesLedgerKey])
		peers, err := e.MessageHub.ListPeers()
		if err != nil {
			return err
		}
		onChainNodes := len(peers)
		p2pPeers := len(e.Host().Network().Peerstore().Peers())
		nodeID := e.Host().ID().String()

		blockchain := ledger.Index()

		return c.JSON(http.StatusOK, types.Summary{
			Files:        files,
			Machines:     machines,
			Users:        users,
			Services:     services,
			BlockChain:   blockchain,
			OnChainNodes: onChainNodes,
			Peers:        p2pPeers,
			NodeID:       nodeID,
		})
	})

	// 机器列表端点
	ec.GET(MachineURL, func(c echo.Context) error {
		list := []*apiTypes.Machine{}

		online := services.AvailableNodes(ledger, 20*time.Minute)

		for _, v := range ledger.CurrentData()[protocol.MachinesLedgerKey] {
			machine := &types.Machine{}
			v.Unmarshal(machine)
			m := &apiTypes.Machine{Machine: *machine}
			// 检查连接状态
			if e.Host().Network().Connectedness(peer.ID(machine.PeerID)) == network.Connected {
				m.Connected = true
			}
			peers, err := e.MessageHub.ListPeers()
			if err != nil {
				return err
			}
			// 检查是否在链上
			for _, p := range peers {
				if p.String() == machine.PeerID {
					m.OnChain = true
				}
			}
			// 检查是否在线
			for _, a := range online {
				if a == machine.PeerID {
					m.Online = true
				}
			}
			list = append(list, m)

		}

		return c.JSON(http.StatusOK, list)
	})

	// 节点列表端点
	ec.GET(NodesURL, func(c echo.Context) error {
		list := []apiTypes.Peer{}
		peers, err := e.MessageHub.ListPeers()
		if err != nil {
			return err
		}

		// 从服务中汇总状态
		online := services.AvailableNodes(ledger, 10*time.Minute)
		p := map[string]interface{}{}

		for _, v := range online {
			p[v] = nil
		}

		for _, v := range peers {
			_, exists := p[v.String()]
			if !exists {
				p[v.String()] = nil
			}
		}

		for id, _ := range p {
			list = append(list, apiTypes.Peer{ID: id, Online: true})
		}

		return c.JSON(http.StatusOK, list)
	})

	// 对等存储端点
	ec.GET(PeerstoreURL, func(c echo.Context) error {
		list := []apiTypes.Peer{}
		for _, v := range e.Host().Network().Peerstore().Peers() {
			list = append(list, apiTypes.Peer{ID: v.String()})
		}
		return c.JSON(http.StatusOK, list)
	})

	// 用户列表端点
	ec.GET(UsersURL, func(c echo.Context) error {
		user := []*types.User{}
		for _, v := range ledger.CurrentData()[protocol.UsersLedgerKey] {
			u := &types.User{}
			v.Unmarshal(u)
			user = append(user, u)
		}
		return c.JSON(http.StatusOK, user)
	})

	// 服务列表端点
	ec.GET(ServiceURL, func(c echo.Context) error {
		list := []*types.Service{}
		for _, v := range ledger.CurrentData()[protocol.ServicesLedgerKey] {
			srvc := &types.Service{}
			v.Unmarshal(srvc)
			list = append(list, srvc)
		}
		return c.JSON(http.StatusOK, list)
	})

	// 静态资源处理
	ec.GET("/*", echo.WrapHandler(http.StripPrefix("/", assetHandler)))

	// 区块链端点
	ec.GET(BlockchainURL, func(c echo.Context) error {
		return c.JSON(http.StatusOK, ledger.LastBlock())
	})

	// 账本端点
	ec.GET(LedgerURL, func(c echo.Context) error {
		return c.JSON(http.StatusOK, ledger.CurrentData())
	})

	// 获取指定存储桶和键的数据
	ec.GET(fmt.Sprintf("%s/:bucket/:key", LedgerURL), func(c echo.Context) error {
		bucket := c.Param("bucket")
		key := c.Param("key")
		return c.JSON(http.StatusOK, ledger.CurrentData()[bucket][key])
	})

	// 获取指定存储桶的数据
	ec.GET(fmt.Sprintf("%s/:bucket", LedgerURL), func(c echo.Context) error {
		bucket := c.Param("bucket")
		return c.JSON(http.StatusOK, ledger.CurrentData()[bucket])
	})

	announcing := struct{ State string }{"Announcing"}

	// 存储任意数据
	ec.PUT(fmt.Sprintf("%s/:bucket/:key/:value", LedgerURL), func(c echo.Context) error {
		bucket := c.Param("bucket")
		key := c.Param("key")
		value := c.Param("value")

		ledger.Persist(context.Background(), defaultInterval, timeout, bucket, key, value)
		return c.JSON(http.StatusOK, announcing)
	})

	// DNS 记录列表端点
	ec.GET(DNSURL, func(c echo.Context) error {
		res := []apiTypes.DNS{}
		for r, e := range ledger.CurrentData()[protocol.DNSKey] {
			var t types.DNS
			e.Unmarshal(&t)
			d := map[string]string{}

			for k, v := range t {
				d[dns.TypeToString[uint16(k)]] = v
			}

			res = append(res,
				apiTypes.DNS{
					Regex:   r,
					Records: d,
				})
		}
		return c.JSON(http.StatusOK, res)
	})

	// 发布 DNS 记录
	ec.POST(DNSURL, func(c echo.Context) error {
		d := new(apiTypes.DNS)
		if err := c.Bind(d); err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}

		entry := make(types.DNS)
		for r, e := range d.Records {
			entry[dns.Type(dns.StringToType[r])] = e
		}
		services.PersistDNSRecord(context.Background(), ledger, defaultInterval, timeout, d.Regex, entry)
		return c.JSON(http.StatusOK, announcing)
	})

	// 从账本删除数据
	ec.DELETE(fmt.Sprintf("%s/:bucket", LedgerURL), func(c echo.Context) error {
		bucket := c.Param("bucket")

		ledger.AnnounceDeleteBucket(context.Background(), defaultInterval, timeout, bucket)
		return c.JSON(http.StatusOK, announcing)
	})

	// 删除指定存储桶和键的数据
	ec.DELETE(fmt.Sprintf("%s/:bucket/:key", LedgerURL), func(c echo.Context) error {
		bucket := c.Param("bucket")
		key := c.Param("key")

		ledger.AnnounceDeleteBucketKey(context.Background(), defaultInterval, timeout, bucket, key)
		return c.JSON(http.StatusOK, announcing)
	})

	ec.HideBanner = true

	// 启动服务器
	if err := ec.Start(l); err != nil && err != http.ErrServerClosed {
		return err
	}

	// 优雅关闭
	go func() {
		<-ctx.Done()
		ct, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		ec.Shutdown(ct)
		cancel()
	}()

	return nil
}
