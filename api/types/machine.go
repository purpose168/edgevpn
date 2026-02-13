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

package types

import "github.com/mudler/edgevpn/pkg/types"

// Machine 表示一个机器实例，扩展了基础机器类型
type Machine struct {
	types.Machine       // 基础机器类型
	Connected    bool   // 是否已连接
	OnChain      bool   // 是否在链上
	Online       bool   // 是否在线
}
