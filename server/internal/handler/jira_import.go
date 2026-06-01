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
	"log-tools/server/internal/pkg/multivolume"

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

type jiraImportJob struct {
	label string
	parts []jiraAttachmentIn
}

func buildJiraImportJobs(attachments []jiraAttachmentIn) ([]jiraImportJob, error) {
	if len(attachments) == 0 {
		return nil, nil
	}
	names := make([]string, len(attachments))
	byName := make(map[string]jiraAttachmentIn, len(attachments))
	for i, a := range attachments {
		names[i] = a.Filename
		byName[a.Filename] = a
	}
	groups, standalone := multivolume.GroupFilenames(names)
	var jobs []jiraImportJob
	for _, g := range groups {
		if !g.IsMultiVolume() {
			return nil, multivolume.VolumeIncompleteError(g)
		}
		job := jiraImportJob{label: g.DisplayName()}
		for _, p := range g.Parts {
			att, ok := byName[p.Filename]
			if !ok {
				return nil, fmt.Errorf("分卷 %s 缺失", p.Filename)
			}
			job.parts = append(job.parts, att)
		}
		jobs = append(jobs, job)
	}
	for _, fn := range standalone {
		jobs = append(jobs, jiraImportJob{
			label: fn,
			parts: []jiraAttachmentIn{byName[fn]},
		})
	}
	return jobs, nil
}

func (h *JiraHandler) importAttachments(
	ctx context.Context,
	deviceID string,
	attachments []jiraAttachmentIn,
	emit jiraProgressEmit,
) ([]string, error) {
	jobs, err := buildJiraImportJobs(attachments)
	if err != nil {
		return nil, err
	}
	dir := h.cfg.Storage.UploadDir
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	total := len(jobs)
	var fileIDs []string
	var lastErr string
	for i, job := range jobs {
		var ids []string
		var impErr error
		if len(job.parts) == 1 {
			ids, impErr = h.importSingleAttachment(ctx, deviceID, dir, job.parts[0], i, total, emit)
		} else {
			ids, impErr = h.importVolumeAttachmentJob(ctx, deviceID, dir, job, i, total, emit)
		}
		if impErr != nil {
			lastErr = impErr.Error()
			log.Printf("[jira] import %s: %v", job.label, impErr)
			continue
		}
		fileIDs = append(fileIDs, ids...)
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

func (h *JiraHandler) importSingleAttachment(
	ctx context.Context,
	deviceID, dir string,
	att jiraAttachmentIn,
	idx, total int,
	emit jiraProgressEmit,
) ([]string, error) {
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
		return nil, err
	}
	if err := h.storage.ValidateFile(int64(len(data)), att.Filename); err != nil {
		return nil, err
	}
	finalPath := filepath.Join(dir, "jira_"+filepath.Base(att.Filename))
	if err := os.WriteFile(finalPath, data, 0o644); err != nil {
		return nil, err
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
		_ = os.Remove(finalPath)
		return nil, err
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("压缩包内未解析到可导入的日志文件")
	}
	emit(JiraProgressEvent{
		Percent:  jiraOverallPercent(idx+1, total, "extract", 100),
		Current:  idx + 1,
		Total:    total,
		Filename: att.Filename,
		Phase:    "extract",
		Message:  "本附件处理完成",
	})
	return ids, nil
}

func (h *JiraHandler) importVolumeAttachmentJob(
	ctx context.Context,
	deviceID, dir string,
	job jiraImportJob,
	idx, total int,
	emit jiraProgressEmit,
) ([]string, error) {
	volDir, err := os.MkdirTemp(dir, "jira-vol-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(volDir)

	partTotal := len(job.parts)
	for pi, att := range job.parts {
		emit(JiraProgressEvent{
			Percent:  jiraOverallPercent(idx, total, "download", pi*100/partTotal),
			Current:  idx + 1,
			Total:    total,
			Filename: job.label,
			Phase:    "download",
			Message:  fmt.Sprintf("正在下载分卷 %d/%d…", pi+1, partTotal),
		})
		data, err := h.client.DownloadAttachmentWithProgress(att.ContentURL, att.ID, func(done, size int64) {
			pct := 0
			if size > 0 {
				pct = int(done * 100 / size)
			} else if done > 0 {
				pct = 50
			}
			inner := (pi*100 + pct) / partTotal
			emit(JiraProgressEvent{
				Percent:  jiraOverallPercent(idx, total, "download", inner),
				Current:  idx + 1,
				Total:    total,
				Filename: job.label,
				Phase:    "download",
				Message:  fmt.Sprintf("正在下载分卷 %d/%d…", pi+1, partTotal),
			})
		})
		if err != nil {
			return nil, err
		}
		if err := h.storage.ValidateFile(int64(len(data)), att.Filename); err != nil {
			return nil, err
		}
		partPath := filepath.Join(volDir, filepath.Base(att.Filename))
		if err := os.WriteFile(partPath, data, 0o644); err != nil {
			return nil, err
		}
	}

	var partPaths []string
	for _, att := range job.parts {
		partPaths = append(partPaths, filepath.Join(volDir, filepath.Base(att.Filename)))
	}
	if len(partPaths) == 0 {
		return nil, fmt.Errorf("分卷压缩包识别失败")
	}

	emit(JiraProgressEvent{
		Percent:  jiraOverallPercent(idx, total, "extract", 0),
		Current:  idx + 1,
		Total:    total,
		Filename: job.label,
		Phase:    "extract",
		Message:  "正在合并分卷并解压…",
	})
	ids, err := h.storage.ImportMultiVolumeArchive(ctx, deviceID, job.label, partPaths)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("压缩包内未解析到可导入的日志文件")
	}
	emit(JiraProgressEvent{
		Percent:  jiraOverallPercent(idx+1, total, "extract", 100),
		Current:  idx + 1,
		Total:    total,
		Filename: job.label,
		Phase:    "extract",
		Message:  "本分卷压缩包处理完成",
	})
	return ids, nil
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
