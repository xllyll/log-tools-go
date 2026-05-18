package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type LogQueryRequest struct {
	FileIDs string `json:"file_ids"`
	Limit   int    `json:"limit"`
	Offset  int    `json:"offset"`
	Module  string `json:"module"`
}

func main() {
	url := "http://127.0.0.1:4080/api/logs"

	// 测试请求参数
	reqBody := LogQueryRequest{
		FileIDs: "bfacce6d,6b757bb2,26c4dfdb,e0ae40ce,aac0c932,a4882d15,b3f61e1e,2318833a",
		Limit:   100,
		Offset:  0,
		Module:  "DeviceService",
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		fmt.Printf("JSON序列化失败: %v\n", err)
		return
	}

	fmt.Println("===========================================")
	fmt.Println("日志查询性能测试")
	fmt.Println("===========================================")
	fmt.Printf("URL: %s\n", url)
	fmt.Printf("请求参数:\n")
	fmt.Printf("  - 文件数量: 8个\n")
	fmt.Printf("  - 模块: %s\n", reqBody.Module)
	fmt.Printf("  - 分页: limit=%d, offset=%d\n", reqBody.Limit, reqBody.Offset)
	fmt.Println("===========================================\n")

	// 执行多次测试取平均值
	totalTime := time.Duration(0)
	testCount := 5

	for i := 1; i <= testCount; i++ {
		fmt.Printf("测试 %d/%d ... ", i, testCount)

		startTime := time.Now()

		resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			fmt.Printf("请求失败: %v\n", err)
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("读取响应失败: %v\n", err)
			continue
		}

		elapsed := time.Since(startTime)
		totalTime += elapsed

		// 解析响应获取记录数
		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err == nil {
			if data, ok := result["data"].([]interface{}); ok {
				fmt.Printf("耗时: %v, 返回记录数: %d\n", elapsed, len(data))
			} else {
				fmt.Printf("耗时: %v\n", elapsed)
			}
		}

		// 间隔一下，避免连续请求
		if i < testCount {
			time.Sleep(500 * time.Millisecond)
		}
	}

	avgTime := totalTime / time.Duration(testCount)
	fmt.Println("\n===========================================")
	fmt.Printf("平均响应时间: %v\n", avgTime)

	if avgTime < 200*time.Millisecond {
		fmt.Println("✅ 性能优秀！(< 200ms)")
	} else if avgTime < 500*time.Millisecond {
		fmt.Println("✅ 性能良好 (< 500ms)")
	} else if avgTime < 1000*time.Millisecond {
		fmt.Println("⚠️  性能一般 (< 1s)，建议进一步优化")
	} else {
		fmt.Println("❌ 性能较差 (> 1s)，请检查索引是否正确创建")
	}
	fmt.Println("===========================================")
}
