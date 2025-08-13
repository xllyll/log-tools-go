package xtime

import (
	"fmt"
	"testing"
)

func TestParseTime(t *testing.T) {
	//now := time.Now()
	//
	//// 现在可以用"熟悉"的格式了！
	//fmt.Println(FormatTime(now, "YYYY-MM-dd HH:mm:ss")) // 2025-08-06 11:20:30
	//fmt.Println(FormatTime(now, "dd/MM/YYYY"))          // 06/08/2025
	//fmt.Println(FormatTime(now, "hh:mm:ss a"))          // 11:20:30 AM
	//fmt.Println(FormatTime(now, "YYYY-MM-dd"))          // 2025-08-06
	//fmt.Println(FormatTime(now, "January 2, 2006"))     // August 6, 2025（Go 原生也支持）

	// 现在可以用"熟悉"的格式了！
	fmt.Println(FormatConvert("YYYY-MM-dd HH:mm:ss"))     // 2025-08-06 11:20:30
	fmt.Println(FormatConvert("dd/MM/YYYY"))              // 06/08/2025
	fmt.Println(FormatConvert("hh:mm:ss a"))              // 11:20:30 AM
	fmt.Println(FormatConvert("YYYY-MM-dd"))              // 2025-08-06
	fmt.Println(FormatConvert("January 2, 2006-YYYY/06")) // August 6, 2025（Go 原生也支持）
}
