package xtime

import (
	"strings"
	"time"
)

func FormatConvert(layout string) string {
	// 定义转换映射：将常见格式转为 Go 的格式
	replacements := map[string]string{
		"YYYY":    "2006",
		"YY":      "06",
		"MM":      "01",  // 月份
		"Jan":     "Jan", // 英文月份缩写（Go 原生支持）
		"January": "January",
		"dd":      "02",   // 日期
		"HH":      "15",   // 24小时制
		"hh":      "03",   // 12小时制
		"mm":      "04",   // 分钟
		"ss":      "05",   // 秒
		"SSS":     ".000", // 毫秒（Go 用 .000 表示）
		"a":       "PM",   // 上午/下午
	}

	// 按长度排序键，避免部分匹配错误（比如 MM 被 dd 替换前先匹配了 M）
	// 这里简单按字符串长度降序处理
	keys := []string{"SSS", "YYYY", "HH", "hh", "mm", "ss", "MM", "dd", "a", "Jan", "January", "YY"}

	result := layout
	for _, k := range keys {
		v := replacements[k]
		result = strings.ReplaceAll(result, k, v)
	}
	return result
}

// FormatTime 使用类似 Java 的格式化字符串来格式化时间
func FormatTime(t time.Time, layout string) string {
	return t.Format(FormatConvert(layout))
}

// ParseTime 使用类似 Java 的格式化字符串来解析时间
func ParseTime(timeStr, layout string) (time.Time, error) {
	return time.Parse(FormatConvert(layout), timeStr)
}
