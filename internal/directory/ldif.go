package directory

import (
	"fmt"
	"strings"
)

// ParseLDIFEntry 解析"单条目 LDIF"（R5.3 专家模式粘贴建条目）。
// 支持续行（前导空格）与 # 注释；base64(::) 值不支持并明确报错。
// 返回可直接交给 Create 的请求。
func ParseLDIFEntry(text string) (*CreateRequest, error) {
	var logical []string // 折行后的逻辑行
	for _, raw := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		if strings.HasPrefix(raw, " ") || strings.HasPrefix(raw, "\t") {
			if len(logical) == 0 {
				return nil, fmt.Errorf("续行不能出现在开头: %q", raw)
			}
			logical[len(logical)-1] += strings.TrimSpace(raw)
			continue
		}
		logical = append(logical, raw)
	}
	req := &CreateRequest{Attrs: map[string][]string{}}
	sawDN := false
	for _, line := range logical {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		i := strings.Index(line, ":")
		if i <= 0 {
			return nil, fmt.Errorf("无法解析的行（应为 属性: 值）: %q", line)
		}
		attr := strings.TrimSpace(line[:i])
		val := strings.TrimSpace(line[i+1:])
		if strings.HasPrefix(val, ":") {
			return nil, fmt.Errorf("暂不支持 base64(::) 值: %s", attr)
		}
		switch strings.ToLower(attr) {
		case "version":
			continue
		case "dn":
			if sawDN {
				return nil, fmt.Errorf("出现多个 dn 行")
			}
			req.DN = val
			sawDN = true
		case "objectclass":
			req.Classes = append(req.Classes, val)
		case "changetype":
			return nil, fmt.Errorf("仅支持新建条目（changetype=%s 不支持）", val)
		default:
			req.Attrs[attr] = append(req.Attrs[attr], val)
		}
	}
	if !sawDN || req.DN == "" {
		return nil, fmt.Errorf("缺少 dn 行")
	}
	if len(req.Classes) == 0 {
		return nil, fmt.Errorf("缺少 objectClass 行")
	}
	return req, nil
}
