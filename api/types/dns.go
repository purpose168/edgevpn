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

// DNS 表示 DNS 记录配置
type DNS struct {
	Regex   string            // 正则表达式匹配模式
	Records map[string]string // DNS 记录映射
}
