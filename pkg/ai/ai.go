package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"io"
	"log"
	"net/http"
	"os"
)

// 定义请求和响应的结构体
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type RequestBody struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type Choice struct {
	Message Message `json:"message"`
}

type ResponseBody struct {
	Choices []Choice `json:"choices"`
}

var _qwen_model = "qwen3-coder-flash"

func Qwen3Chat(apiKey string, model *string, msg string) (*string, error) {
	// 设置 API Key 和 URL
	if model != nil {
		model = &_qwen_model
	}
	url := "https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions"

	// 构建请求体
	requestBody := RequestBody{
		Model: _qwen_model,
		Messages: []Message{
			{Role: "assistant", Content: "你好啊，我是通义千问。"},
			{Role: "user", Content: msg},
		},
	}

	// 将请求体序列化为 JSON
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		fmt.Printf("序列化请求体失败: %v\n", err)
		return nil, err
	}

	// 创建 HTTP 请求
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		fmt.Printf("创建请求失败: %v\n", err)
		return nil, err
	}

	// 设置请求头
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("发送请求失败: %v\n", err)
		return nil, err
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("读取响应失败: %v\n", err)
		return nil, err
	}

	// 检查 HTTP 状态码
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("请求失败，状态码: %d, 响应: %s\n", resp.StatusCode, string(body))
		return nil, fmt.Errorf("请求失败，状态码:%s", resp.StatusCode)
	}

	// 解析响应
	var responseBody ResponseBody
	err = json.Unmarshal(body, &responseBody)
	if err != nil {
		fmt.Printf("解析响应失败: %v\n", err)
		return nil, err
	}
	aiRes := "未收到有效回复"
	// 输出助手的回复
	if len(responseBody.Choices) > 0 {
		aiRes = responseBody.Choices[0].Message.Content
		fmt.Println("助手回复:", aiRes)
	} else {
		fmt.Println("未收到有效回复")
	}
	return &aiRes, nil
}

// ai/stream.go 或 ai/ai.go 中添加
func Qwen3ChatStream(apiKey, model, userMsg string, writer http.ResponseWriter) error {
	url := "https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions"

	if model == "" {
		model = _qwen_model // 假设你有默认 model
	}

	requestBody := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "assistant", "content": "你好啊，我是通义千问。"},
			{"role": "user", "content": userMsg},
		},
		"stream": true,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("序列化请求体失败: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("调用AI失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		//读取 body 的值
		body, _ := io.ReadAll(resp.Body)
		respJson := map[string]interface{}{}
		err = json.Unmarshal(body, &respJson)
		if err != nil {
			return fmt.Errorf("解析响应失败: %v", err)
		}
		fmt.Printf("调用AI失败: %s", respJson)
		errMsg, ok := respJson["error"].(map[string]interface{})
		if ok {
			return fmt.Errorf("%s", errMsg["message"])
		}
		return fmt.Errorf("%s", respJson)
	}

	// 使用 bufio 逐行读取 SSE
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		log.Println("接收到数据:", line)
		if len(line) > 6 && line[:6] == "data: " {
			data := line[6:]
			if data == "[DONE]" {
				break
			}
			// 解析 content
			var chunk map[string]interface{}
			if json.Unmarshal([]byte(data), &chunk); err != nil {
				continue
			}
			if choices, ok := chunk["choices"].([]interface{}); ok && len(choices) > 0 {
				if choice, ok := choices[0].(map[string]interface{}); ok {
					if delta, ok := choice["delta"].(map[string]interface{}); ok {
						if content, ok := delta["content"].(string); ok && content != "" {
							// 发送到客户端
							resData := map[string]interface{}{
								"type": "stream",
								"msg":  content,
							}
							jsonData, _ := json.Marshal(resData)
							fmt.Fprintf(writer, "data: %s\n\n", jsonData)
							writer.(http.Flusher).Flush() // 关键：强制刷新
						}
					}
				}
			}
		}
	}
	return nil
}

func QwenChat(msg string) {
	client := openai.NewClient(
		option.WithAPIKey(os.Getenv("sk-xxxxxxxxxxxxxxxxxxxxxxxxxxx")),
		option.WithBaseURL("https://dashscope.aliyuncs.com/compatible-mode/v1/"),
	)
	chatCompletion, err := client.Chat.Completions.New(
		context.TODO(), openai.ChatCompletionNewParams{
			Messages: openai.F(
				[]openai.ChatCompletionMessageParamUnion{
					openai.UserMessage(msg),
				},
			),
			Model: openai.F("qwen3-coder-flash"),
		},
	)
	if err != nil {
		panic(err.Error())
	}
	println(chatCompletion.Choices[0].Message.Content)
}
