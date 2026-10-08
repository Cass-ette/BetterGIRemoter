package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	staticfiles "BetterGIRemoter/static"
	"BetterGIRemoter/store"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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

// probeBetterGI 用短超时探测 BetterGI API 是否就绪
func (h *Handler) probeBetterGI() error {
	req, err := http.NewRequest("GET", h.bettergiURL+"/api/status", nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-API-Token", h.bettergiToken)
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// ensureBetterGI 确保 BetterGI 正在运行且 API 就绪；未运行则自动启动并等待就绪
func (h *Handler) ensureBetterGI() error {
	if err := h.probeBetterGI(); err == nil {
		return nil
	}

	exe := os.Getenv("BETTERGI_EXE")
	if exe == "" {
		return errors.New("BetterGI 未运行，且未配置 BETTERGI_EXE 环境变量，无法自动启动")
	}
	if _, statErr := os.Stat(exe); statErr != nil {
		return fmt.Errorf("BETTERGI_EXE 路径无效: %s", exe)
	}

	cmd := exec.Command(exe)
	cmd.Dir = filepath.Dir(exe)
	if startErr := cmd.Start(); startErr != nil {
		return fmt.Errorf("启动 BetterGI 失败: %w", startErr)
	}

	// 等待 BetterGI 初始化完成（加载模型等需要时间），最多 90 秒
	for i := 0; i < 90; i++ {
		time.Sleep(1 * time.Second)
		if err := h.probeBetterGI(); err == nil {
			return nil
		}
	}
	return errors.New("BetterGI 进程已启动，但 API 在 90 秒内未就绪，请检查 BetterGI 是否正常启动")
}

// StartBetterGI 显式启动 BetterGI（幂等：已运行则直接返回）
func (h *Handler) StartBetterGI(c *gin.Context) {
	if err := h.ensureBetterGI(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "BetterGI 已就绪"})
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

	if err := h.ensureBetterGI(); err != nil {
		h.store.Update(taskID, "failed", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

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

	if err := h.ensureBetterGI(); err != nil {
		h.store.Update(taskID, "failed", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

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

	if err := h.ensureBetterGI(); err != nil {
		h.store.Update(taskID, "failed", err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

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
	data, err := staticfiles.FS.ReadFile("index.html")
	if err != nil {
		c.String(http.StatusInternalServerError, "index.html not found")
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", data)
}
