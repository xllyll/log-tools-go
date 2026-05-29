package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"log-tools/server/internal/model"

	"github.com/gin-gonic/gin"
)

type jiraAttachmentIn struct {
	ID         string `json:"id"`
	Filename   string `json:"filename"`
	ContentURL string `json:"content_url"`
}

type jiraImportRequest struct {
	IssueKey    string             `json:"issue_key"`
	Attachments []jiraAttachmentIn `json:"attachments"`
}

// JiraProgressEvent SSE 进度事件
type JiraProgressEvent struct {
	Percent  int    `json:"percent"`
	Current  int    `json:"current"`
	Total    int    `json:"total"`
	Filename string `json:"filename,omitempty"`
	Phase    string `json:"phase"` // download | extract | done | error
	Message  string `json:"message,omitempty"`
}

type jiraProgressEmit func(ev JiraProgressEvent)

func jiraOverallPercent(fileIndex, fileTotal int, phase string, phasePercent int) int {
	if fileTotal <= 0 {
		return 0
	}
	if phasePercent < 0 {
		phasePercent = 0
	}
	if phasePercent > 100 {
		phasePercent = 100
	}
	w := 100.0 / float64(fileTotal)
	base := float64(fileIndex) * w
	var inFile float64
	switch phase {
	case "download":
		inFile = w * 0.7 * float64(phasePercent) / 100
	case "extract":
		inFile = w*0.7 + w*0.3*float64(phasePercent)/100
	default:
		inFile = w
	}
	p := int(base + inFile)
	if p > 100 {
		return 100
	}
	return p
}

func writeJiraSSE(c *gin.Context, flusher http.Flusher, event string, payload any) {
	b, _ := json.Marshal(payload)
	_, _ = fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, b)
	flusher.Flush()
}

func (h *JiraHandler) importAttachments(
	ctx context.Context,
	deviceID string,
	attachments []jiraAttachmentIn,
	emit jiraProgressEmit,
) ([]string, error) {
	dir := h.cfg.Storage.UploadDir
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	total := len(attachments)
	var fileIDs []string
	var lastErr string
	for i, att := range attachments {
		idx := i
		emit(JiraProgressEvent{
			Percent:  jiraOverallPercent(idx, total, "download", 0),
			Current:  idx + 1,
			Total:    total,
			Filename: att.Filename,
			Phase:    "download",
			Message:  "正在从 Jira 下载…",
		})
		data, err := h.client.DownloadAttachmentWithProgress(att.ContentURL, att.ID, func(done, size int64) {
			pct := 0
			if size > 0 {
				pct = int(done * 100 / size)
			} else if done > 0 {
				pct = 50
			}
			emit(JiraProgressEvent{
				Percent:  jiraOverallPercent(idx, total, "download", pct),
				Current:  idx + 1,
				Total:    total,
				Filename: att.Filename,
				Phase:    "download",
				Message:  "正在从 Jira 下载…",
			})
		})
		if err != nil {
			lastErr = err.Error()
			log.Printf("[jira] download %s: %v", att.Filename, err)
			continue
		}
		if err := h.storage.ValidateFile(int64(len(data)), att.Filename); err != nil {
			lastErr = err.Error()
			log.Printf("[jira] skip %s: %v", att.Filename, err)
			continue
		}
		finalPath := filepath.Join(dir, "jira_"+filepath.Base(att.Filename))
		if err := os.WriteFile(finalPath, data, 0o644); err != nil {
			lastErr = err.Error()
			continue
		}
		emit(JiraProgressEvent{
			Percent:  jiraOverallPercent(idx, total, "extract", 0),
			Current:  idx + 1,
			Total:    total,
			Filename: att.Filename,
			Phase:    "extract",
			Message:  "正在解压并登记日志…",
		})
		ids, err := h.storage.ImportSavedFile(ctx, deviceID, finalPath, att.Filename)
		if err != nil {
			lastErr = err.Error()
			_ = os.Remove(finalPath)
			log.Printf("[jira] import %s: %v", att.Filename, err)
			continue
		}
		if len(ids) == 0 {
			lastErr = "压缩包内未解析到可导入的日志文件"
			log.Printf("[jira] no logs from %s", att.Filename)
			continue
		}
		fileIDs = append(fileIDs, ids...)
		emit(JiraProgressEvent{
			Percent:  jiraOverallPercent(idx+1, total, "extract", 100),
			Current:  idx + 1,
			Total:    total,
			Filename: att.Filename,
			Phase:    "extract",
			Message:  "本附件处理完成",
		})
	}
	if len(fileIDs) == 0 {
		msg := "未能导入任何附件"
		if lastErr != "" {
			msg = msg + ": " + lastErr
		}
		return nil, fmt.Errorf("%s", msg)
	}
	return fileIDs, nil
}

// ImportStream 通过 SSE 推送下载/解压进度
func (h *JiraHandler) ImportStream(c *gin.Context) {
	deviceID := GetDeviceID(c)
	var req jiraImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Success: false, Error: err.Error()})
		return
	}
	if len(req.Attachments) == 0 {
		c.JSON(http.StatusBadRequest, model.APIResponse{Success: false, Error: "请选择至少一个附件"})
		return
	}

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, model.APIResponse{Success: false, Error: "streaming not supported"})
		return
	}
	c.Header("Content-Type", "text/event-stream; charset=utf-8")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)

	emit := func(ev JiraProgressEvent) {
		writeJiraSSE(c, flusher, "progress", ev)
	}

	ctx := c.Request.Context()
	fileIDs, err := h.importAttachments(ctx, deviceID, req.Attachments, emit)
	if err != nil {
		writeJiraSSE(c, flusher, "error", JiraProgressEvent{
			Phase:   "error",
			Message: err.Error(),
		})
		return
	}
	writeJiraSSE(c, flusher, "done", map[string]any{
		"file_ids": fileIDs,
		"message":  "jira 附件已导入",
	})
}
