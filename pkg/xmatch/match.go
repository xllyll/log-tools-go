package xmatch

import (
	"fmt"
	"regexp"
)

// 通过匹配规则（正则）配置字符串是否包含的字符串
func Match(pattern string, str string) string {
	if pattern == "" {
		return ""
	}
	r, err := regexp.Compile(pattern)
	if err != nil {
		fmt.Println(err)
		return ""
	}
	match := r.FindStringSubmatch(str)
	if len(match) > 1 {
		return match[1]
	}
	return ""
}
