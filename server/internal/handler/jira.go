package handler

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"log-tools/server/internal/model"
	"log-tools/server/internal/service"

	"github.com/gin-gonic/gin"
)

type JiraHandler struct {
	storage *service.StorageService
}

func NewJiraHandler(storage *service.StorageService) *JiraHandler {
	return &JiraHandler{storage: storage}
}

func (h *JiraHandler) ListAttachments(c *gin.Context) {
	var req model.JiraConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Success: false, Error: err.Error()})
		return
	}
	issueKey := c.Param("key")
	if issueKey == "" {
		c.JSON(http.StatusBadRequest, model.APIResponse{Success: false, Error: "issue key required"})
		return
	}
	url := strings.TrimRight(req.BaseURL, "/") + "/rest/api/2/issue/" + issueKey + "?fields=attachment"
	body, err := jiraGET(url, req.Email, req.APIToken)
	if err != nil {
		c.JSON(http.StatusBadGateway, model.APIResponse{Success: false, Error: err.Error()})
		return
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
		c.JSON(http.StatusBadGateway, model.APIResponse{Success: false, Error: err.Error()})
		return
	}
	var list []model.JiraAttachment
	for _, a := range parsed.Fields.Attachment {
		ext := strings.ToLower(filepath.Ext(a.Filename))
		if ext != ".log" && ext != ".txt" && ext != ".zip" && ext != ".rar" && ext != ".7z" {
			continue
		}
		list = append(list, model.JiraAttachment{
			ID: a.ID, Filename: a.Filename, Size: a.Size, MimeType: a.MimeType, ContentURL: a.Content,
		})
	}
	c.JSON(http.StatusOK, model.APIResponse{Success: true, Data: list})
}

func (h *JiraHandler) Import(c *gin.Context) {
	deviceID := GetDeviceID(c)
	var req struct {
		Config      model.JiraConfig `json:"config"`
		IssueKey    string           `json:"issue_key"`
		Attachments []struct {
			ID       string `json:"id"`
			Filename string `json:"filename"`
			Content  string `json:"content_url"`
		} `json:"attachments"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.APIResponse{Success: false, Error: err.Error()})
		return
	}
	var fileIDs []string
	for _, att := range req.Attachments {
		url := att.Content
		if url == "" {
			url = strings.TrimRight(req.Config.BaseURL, "/") + "/rest/api/2/attachment/" + att.ID
		}
		data, err := jiraGET(url, req.Config.Email, req.Config.APIToken)
		if err != nil {
			continue
		}
		tmp, err := os.CreateTemp("", "jira_*"+filepath.Ext(att.Filename))
		if err != nil {
			continue
		}
		_, _ = tmp.Write(data)
		tmp.Close()
		path := tmp.Name()
		_ = os.Rename(path, path+filepath.Ext(att.Filename))
		finalPath := path + filepath.Ext(att.Filename)
		lf, err := h.storage.StartIngest(deviceID, finalPath)
		if err != nil {
			_ = os.Remove(finalPath)
			continue
		}
		fileIDs = append(fileIDs, lf.ID)
	}
	c.JSON(http.StatusOK, model.UploadResponse{
		Success: true,
		Message: "jira import queued",
		FileIDs: fileIDs,
	})
}

func jiraGET(url, email, token string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	auth := base64.StdEncoding.EncodeToString([]byte(email + ":" + token))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("jira status %d: %s", resp.StatusCode, string(b))
	}
	return io.ReadAll(resp.Body)
}
