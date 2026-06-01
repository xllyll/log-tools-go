package jira

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"log-tools/server/internal/config"
	"log-tools/server/internal/model"
)

type Client struct {
	cfg config.JiraConfig
}

func NewClient(cfg config.JiraConfig) *Client {
	return &Client{cfg: cfg}
}

func (c *Client) enabled() error {
	if !c.cfg.Enabled {
		return fmt.Errorf("jira 未启用，请在服务端 config.yaml 中配置")
	}
	if c.cfg.BaseURL == "" || c.cfg.Email == "" || c.cfg.APIToken == "" {
		return fmt.Errorf("jira 配置不完整，请检查 base_url / email / api_token")
	}
	return nil
}

func (c *Client) ListLogAttachments(issueKey string) ([]model.JiraAttachment, error) {
	if err := c.enabled(); err != nil {
		return nil, err
	}
	issueKey = strings.TrimSpace(issueKey)
	if issueKey == "" {
		return nil, fmt.Errorf("issue key 不能为空")
	}
	url := strings.TrimRight(c.cfg.BaseURL, "/") + "/rest/api/2/issue/" + issueKey + "?fields=attachment"
	body, err := c.get(url)
	if err != nil {
		return nil, err
	}
	var parsed struct {
		Fields struct {
			Attachment []struct {
				ID       string `json:"id"`
				Filename string `json:"filename"`
				Size     int64  `json:"size"`
				MimeType string `json:"mimeType"`
				Content  string `json:"content"`
			} `json:"attachment"`
		} `json:"fields"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	var list []model.JiraAttachment
	for _, a := range parsed.Fields.Attachment {
		ext := strings.ToLower(filepath.Ext(a.Filename))
		if !model.IsLogFileName(a.Filename) && ext != ".zip" && ext != ".rar" && ext != ".7z" {
			continue
		}
		list = append(list, model.JiraAttachment{
			ID: a.ID, Filename: a.Filename, Size: a.Size, MimeType: a.MimeType, ContentURL: a.Content,
		})
	}
	return list, nil
}

// DownloadProgressFunc 下载进度回调；total 未知时为 -1
type DownloadProgressFunc func(downloaded, total int64)

func (c *Client) DownloadAttachment(contentURL, attachmentID string) ([]byte, error) {
	return c.DownloadAttachmentWithProgress(contentURL, attachmentID, nil)
}

func (c *Client) DownloadAttachmentWithProgress(contentURL, attachmentID string, onProgress DownloadProgressFunc) ([]byte, error) {
	if err := c.enabled(); err != nil {
		return nil, err
	}
	url := contentURL
	if url == "" {
		url = strings.TrimRight(c.cfg.BaseURL, "/") + "/rest/api/2/attachment/" + attachmentID
	}
	return c.getWithProgress(url, onProgress)
}

func (c *Client) get(url string) ([]byte, error) {
	return c.getWithProgress(url, nil)
}

type byteCounter struct {
	n          int64
	total      int64
	onProgress DownloadProgressFunc
}

func (c *byteCounter) Write(p []byte) (int, error) {
	c.n += int64(len(p))
	if c.onProgress != nil {
		c.onProgress(c.n, c.total)
	}
	return len(p), nil
}

func (c *Client) getWithProgress(url string, onProgress DownloadProgressFunc) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	auth := base64.StdEncoding.EncodeToString([]byte(c.cfg.Email + ":" + c.cfg.APIToken))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Accept", "*/*")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("jira 请求失败 %d: %s", resp.StatusCode, string(data))
	}
	total := resp.ContentLength
	if onProgress != nil {
		onProgress(0, total)
	}
	counter := &byteCounter{total: total, onProgress: onProgress}
	data, err := io.ReadAll(io.TeeReader(resp.Body, counter))
	if err != nil {
		return nil, err
	}
	if onProgress != nil {
		onProgress(int64(len(data)), total)
	}
	return data, nil
}
