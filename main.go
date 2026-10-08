package main

import (
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"BetterGIRemoter/handler"
	staticfiles "BetterGIRemoter/static"
	"BetterGIRemoter/store"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// BetterGI API 地址和 Token
	bettergiURL := os.Getenv("BETTERGI_URL")
	if bettergiURL == "" {
		bettergiURL = "http://localhost:8080" // BetterGI 默认端口
	}

	bettergiToken := os.Getenv("BETTERGI_TOKEN")
	if bettergiToken == "" {
		log.Println("警告: 未设置 BETTERGI_TOKEN 环境变量")
	}

	s := store.New()
	h := handler.New(s, bettergiURL, bettergiToken)

	r := gin.Default()

	// 管理端点（可通过 ADMIN_TOKEN 环境变量保护）
	admin := r.Group("/admin", h.AdminAuth())
	{
		admin.POST("/start", h.StartTask)
		admin.POST("/stop", h.StopTask)
		admin.GET("/status", h.GetStatus)
		admin.GET("/tasks", h.ListTasks)
		admin.GET("/scripts", h.GetScripts)
		admin.POST("/script/:id/start", h.StartScript)
		admin.GET("/screenshot", h.GetScreenshot)
		admin.GET("/onedragon/configs", h.GetOneDragonConfigs)
		admin.POST("/onedragon/execute/:id", h.ExecuteOneDragon)
		admin.POST("/bettergi/start", h.StartBetterGI)
	}

	// 静态文件（内嵌于二进制）
	sub, _ := fs.Sub(staticfiles.FS, ".")
	r.StaticFS("/static", http.FS(sub))

	// 控制面板
	r.GET("/", h.Dashboard)

	log.Printf("BetterGI Remote 已启动，访问 http://localhost:%s", port)
	log.Printf("BetterGI API 地址: %s", bettergiURL)
	log.Fatal(r.Run(":" + port))
}
