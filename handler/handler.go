package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"BetterGIRemoter/store"
)

type Handler struct {
	store         *store.Store
	bettergiURL   string
	bettergiToken string
}

func New(s *store.Store, url, token string) *Handler {
	return &Handler{
		store:         s,
		bettergiURL:   url,
		bettergiToken: token,
	}
}

// AdminAuth 中间件：通过 ADMIN_TOKEN 环境变量保护管理端点
func (h *Handler) AdminAuth() gin.HandlerFunc {
	adminToken := os.Getenv("ADMIN_TOKEN")
	return func(c *gin.Context) {
		if adminToken == "" {
			c.Next()
			return
		}

		token := c.GetHeader("Authorization")
		if token != "Bearer "+adminToken {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// callBetterGI 调用 BetterGI HTTP API
func (h *Handler) callBetterGI(method, path string) (int, string, error) {
	url := h.bettergiURL + path
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return 0, "", err
	}

	req.Header.Set("X-API-Token", h.bettergiToken)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, "", err
	}

	return resp.StatusCode, string(body), nil
}

// StartTask 启动 BetterGI 任务
func (h *Handler) StartTask(c *gin.Context) {
	taskID := uuid.New().String()
	task := &store.Task{
		ID:        taskID,
		Type:      "start",
		Status:    "running",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	h.store.Add(task)

	statusCode, body, err := h.callBetterGI("POST", "/api/start")
	if err != nil {
		h.store.Update(taskID, "failed", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if statusCode != http.StatusOK {
		h.store.Update(taskID, "failed", fmt.Sprintf("HTTP %d: %s", statusCode, body))
		c.JSON(statusCode, gin.H{"error": body})
		return
	}

	h.store.Update(taskID, "completed", body)
	c.JSON(http.StatusOK, gin.H{
		"task_id": taskID,
		"status":  "completed",
		"result":  body,
	})
}

// StopTask 停止 BetterGI 任务
func (h *Handler) StopTask(c *gin.Context) {
	taskID := uuid.New().String()
	task := &store.Task{
		ID:        taskID,
		Type:      "stop",
		Status:    "running",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	h.store.Add(task)

	statusCode, body, err := h.callBetterGI("POST", "/api/stop")
	if err != nil {
		h.store.Update(taskID, "failed", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if statusCode != http.StatusOK {
		h.store.Update(taskID, "failed", fmt.Sprintf("HTTP %d: %s", statusCode, body))
		c.JSON(statusCode, gin.H{"error": body})
		return
	}

	h.store.Update(taskID, "completed", body)
	c.JSON(http.StatusOK, gin.H{
		"task_id": taskID,
		"status":  "completed",
		"result":  body,
	})
}

// GetStatus 获取 BetterGI 状态
func (h *Handler) GetStatus(c *gin.Context) {
	statusCode, body, err := h.callBetterGI("GET", "/api/status")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if statusCode != http.StatusOK {
		c.JSON(statusCode, gin.H{"error": body})
		return
	}

	// 解析 JSON 响应
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(body), &result); err != nil {
		c.JSON(http.StatusOK, gin.H{"raw": body})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ListTasks 列出最近的任务
func (h *Handler) ListTasks(c *gin.Context) {
	tasks := h.store.Recent(50)
	c.JSON(http.StatusOK, gin.H{"tasks": tasks})
}

// GetScripts 获取脚本列表
func (h *Handler) GetScripts(c *gin.Context) {
	statusCode, body, err := h.callBetterGI("GET", "/api/scripts")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if statusCode != http.StatusOK {
		c.JSON(statusCode, gin.H{"error": body})
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(body), &result); err != nil {
		c.JSON(http.StatusOK, gin.H{"raw": body})
		return
	}

	c.JSON(http.StatusOK, result)
}

// StartScript 启动指定脚本
func (h *Handler) StartScript(c *gin.Context) {
	scriptID := c.Param("id")
	if scriptID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "脚本ID不能为空"})
		return
	}

	taskID := uuid.New().String()
	task := &store.Task{
		ID:        taskID,
		Type:      "script",
		Status:    "running",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	h.store.Add(task)

	statusCode, body, err := h.callBetterGI("POST", "/api/script/"+scriptID+"/start")
	if err != nil {
		h.store.Update(taskID, "failed", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if statusCode != http.StatusOK {
		h.store.Update(taskID, "failed", fmt.Sprintf("HTTP %d: %s", statusCode, body))
		c.JSON(statusCode, gin.H{"error": body})
		return
	}

	h.store.Update(taskID, "completed", body)
	c.JSON(http.StatusOK, gin.H{
		"task_id": taskID,
		"status":  "completed",
		"result":  body,
	})
}

// GetScreenshot 获取游戏截图
func (h *Handler) GetScreenshot(c *gin.Context) {
	statusCode, body, err := h.callBetterGI("GET", "/api/screenshot")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if statusCode != http.StatusOK {
		c.JSON(statusCode, gin.H{"error": body})
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(body), &result); err != nil {
		c.JSON(http.StatusOK, gin.H{"raw": body})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetOneDragonConfigs 获取一条龙配置列表
func (h *Handler) GetOneDragonConfigs(c *gin.Context) {
	statusCode, body, err := h.callBetterGI("GET", "/api/onedragon/configs")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if statusCode != http.StatusOK {
		c.JSON(statusCode, gin.H{"error": body})
		return
	}

	var result map[string]interface{}
	if err := json.Unmarshal([]byte(body), &result); err != nil {
		c.JSON(http.StatusOK, gin.H{"raw": body})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ExecuteOneDragon 执行一条龙配置
func (h *Handler) ExecuteOneDragon(c *gin.Context) {
	configName := c.Param("id")
	if configName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "配置名不能为空"})
		return
	}

	taskID := uuid.New().String()
	task := &store.Task{
		ID:        taskID,
		Type:      "onedragon",
		Status:    "running",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	h.store.Add(task)

	statusCode, body, err := h.callBetterGI("POST", "/api/onedragon/execute/"+configName)
	if err != nil {
		h.store.Update(taskID, "failed", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if statusCode != http.StatusOK {
		h.store.Update(taskID, "failed", fmt.Sprintf("HTTP %d: %s", statusCode, body))
		c.JSON(statusCode, gin.H{"error": body})
		return
	}

	h.store.Update(taskID, "completed", body)
	c.JSON(http.StatusOK, gin.H{
		"task_id": taskID,
		"status":  "completed",
		"result":  body,
	})
}

// Dashboard 控制面板页面
func (h *Handler) Dashboard(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{
		"bettergiURL": h.bettergiURL,
	})
}
