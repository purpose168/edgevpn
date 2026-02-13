package discovery

// Ring 环形缓冲区结构体
type Ring struct {
	Data   []string // 数据存储
	Length int      // 环形缓冲区长度
}

// Add 向环形缓冲区添加元素
// 参数 s 为要添加的字符串
func (r *Ring) Add(s string) {
	if len(r.Data) > 0 {
		// 避免最后一项重复
		if r.Data[len(r.Data)-1] == s {
			return
		}
	}

	// 如果超过长度限制，移除第一个元素
	if len(r.Data)+1 > r.Length {
		r.Data = r.Data[1:]
	}
	r.Data = append(r.Data, s)
}
