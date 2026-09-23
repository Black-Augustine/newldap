// uid 自动生成：姓名拼音 → 冲突加序号（新手只需填姓名，账号可留空自动生成）。
package people

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/mozillazg/go-pinyin"

	ldap "github.com/go-ldap/ldap/v3"
)

var nonAlnum = regexp.MustCompile(`[^a-z0-9._-]`)

// GenerateUID 依据姓名生成英文账号：中文转全拼（张伟→zhangwei），
// 英文直接小写去非法字符，空结果回退 "user"。
func GenerateUID(name string) string {
	if name == "" {
		return "user"
	}
	// 已是纯 ASCII：直接规范化
	if isASCII(name) {
		base := nonAlnum.ReplaceAllString(strings.ToLower(name), "")
		if base != "" {
			return base
		}
		return "user"
	}
	// 中文/混合：逐字符取拼音首字符全拼（默认全拼）
	var sb strings.Builder
	for _, r := range name {
		if isASCII(string(r)) {
			sb.WriteRune(r)
			continue
		}
		a := pinyin.NewArgs()
		a.Style = pinyin.Normal
		ps := pinyin.Pinyin(string(r), a)
		if len(ps) > 0 && len(ps[0]) > 0 {
			sb.WriteString(ps[0][0])
		}
	}
	base := nonAlnum.ReplaceAllString(strings.ToLower(sb.String()), "")
	if base == "" {
		return "user"
	}
	return base
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > 127 {
			return false
		}
	}
	return true
}

// UniqueUID 在 searchBase 子树内检查唯一性（全组织范围），冲突时追加 2/3/4…
func (s *Service) UniqueUID(base, searchBase string) (string, error) {
	candidate := base
	for i := 2; ; i++ {
		hits, err := s.C.PagedSearch(searchBase, ldap.ScopeWholeSubtree,
			"(uid="+ldap.EscapeFilter(candidate)+")", []string{"1.1"}, 1)
		if err != nil {
			return "", err
		}
		if len(hits) == 0 {
			return candidate, nil
		}
		candidate = fmt.Sprintf("%s%d", base, i)
		if i > 99 {
			return "", fmt.Errorf("自动生成账号失败：%s 序号耗尽", base)
		}
	}
}
