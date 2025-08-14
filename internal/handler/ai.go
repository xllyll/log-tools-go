package handler

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"log-tools-go/internal/config"
	"log-tools-go/internal/model"
	"log-tools-go/internal/service"
	"log-tools-go/pkg/ai"
	"net/http"
)

type AiHandler struct {
	config  *config.Config
	storage *service.StorageService
	parser  *service.LogParser
}

func NewAiHandler(cfg *config.Config, storage *service.StorageService, parser *service.LogParser) *AiHandler {
	return &AiHandler{
		config:  cfg,
		storage: storage,
		parser:  parser,
	}
}

func (h *AiHandler) AnalysisLog(ctx *gin.Context) {
	req := &model.AnalysisLogRequest{}
	if err := ctx.ShouldBindJSON(req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}
	chatMsg := "分析下这些日志的问题：\n" + req.Logs
	res, err := ai.Qwen3Chat(h.config.AiConfig.ApiKey, &h.config.AiConfig.Model, chatMsg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "分析日志失败: " + err.Error(),
		})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    *res,
	})
}

func (h *AiHandler) AnalysisLogStream(ctx *gin.Context) {
	req := &model.AnalysisLogRequest{}
	if err := ctx.ShouldBindJSON(req); err != nil {
		// ❌ 不要再用 ctx.JSON
		// ✅ 改为通过流输出错误，并结束
		ctx.Header("Content-Type", "text/event-stream")
		ctx.Header("Cache-Control", "no-cache")
		ctx.Header("Connection", "keep-alive")

		// 发送错误并结束
		fmt.Fprintf(ctx.Writer, "data: %s\n\n", fmt.Sprintf(`{"success":false,"error":"%s","type":"error"}`, err.Error()))
		fmt.Fprintf(ctx.Writer, "data: %s\n\n", `{"type":"done"}`)
		ctx.Writer.Flush()
		return
	}

	// 设置流式响应头
	ctx.Header("Content-Type", "text/event-stream")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")
	ctx.Header("Transfer-Encoding", "chunked")

	// 自定义 success 标志开头
	fmt.Fprintf(ctx.Writer, "data: %s\n\n", `{"success":true,"type":"start"}`)
	ctx.Writer.Flush()

	// 构造提示词
	chatMsg := req.Logs

	// 调用流式 AI 接口
	err := ai.Qwen3ChatStream(h.config.AiConfig.ApiKey, h.config.AiConfig.Model, chatMsg, ctx.Writer)
	if err != nil {
		// 发送错误信息（仍走流）
		fmt.Fprintf(ctx.Writer, "data: %s\n\n", fmt.Sprintf(`{"error":"%s","type":"error"}`, err.Error()))
		ctx.Writer.Flush()
		return
	}
	// 结束标记
	fmt.Fprintf(ctx.Writer, "data: %s\n\n", `{"type":"done"}`)
	ctx.Writer.Flush()
}
