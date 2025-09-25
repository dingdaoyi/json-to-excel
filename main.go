package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	"json-to-excel/config"
	"json-to-excel/internal"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	host        = flag.String("host", getEnvOrDefault("HOST", "localhost"), "监听地址")
	port        = flag.String("port", getEnvOrDefault("PORT", "8080"), "监听端口")
	baseURL     = flag.String("base-url", getEnvOrDefault("BASE_URL", ""), "外部访问基础URL")
	downloadDir = getEnvOrDefault("DOWNLOAD_DIR", "./downloads")
)

// getEnvOrDefault 获取环境变量，如果不存在则返回默认值
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvDurationOrDefault 获取环境变量并转换为时间间隔
func getEnvDurationOrDefault(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
		internal.Logger.WithFields(map[string]interface{}{
			"key":     key,
			"value":   value,
			"default": defaultValue,
		}).Warn("无法解析环境变量，使用默认值")
	}
	return defaultValue
}

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "config 服务.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		os.Exit(1)
	}
	flag.Parse()

	addr := fmt.Sprintf("%s:%s", *host, *port)

	e := gin.Default()

	e.Use(gin.Logger())
	e.Use(gin.Recovery())
	e.Use(ErrorHandlerMiddleware())
	e.Use(CORSMiddleware())

	// 文件下载服务
	e.Static("/downloads", downloadDir)
	mux := http.NewServeMux()
	c := internal.Config{
		TempDir:       downloadDir,
		Port:          *port,
		Host:          *host,
		BaseURL:       *baseURL,
		ExpirationDur: getEnvDurationOrDefault("FILE_EXPIRATION", 2*time.Minute),
		CleanupTick:   getEnvDurationOrDefault("CLEANUP_INTERVAL", 30*time.Second),
	}
	mcpHandler := config.NewMCPHandler(c)

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	loggingHandler := config.McpLoggingHandler(mcpHandler)
	mux.Handle("/mcp", loggingHandler)
	mux.Handle("/", e)
	srv := &http.Server{
		Addr:           addr,
		Handler:        mux,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	internal.Logger.WithField("addr", addr).Info("服务启动")
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			internal.Logger.WithError(err).Fatal("服务启动失败")
		}
	}()
	// 等待中断信号
	<-sigChan
	internal.Logger.Info("正在关闭服务...")

	// 然后关闭 Excel 服务
	if err := mcpHandler.Close(); err != nil {
		internal.Logger.WithError(err).Error("Excel service Close")
	}

	internal.Logger.Info("服务已完全关闭")
}

// ErrorHandlerMiddleware 统一错误返回
func ErrorHandlerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) > 0 {
			c.JSON(-1, gin.H{
				"code":  500,
				"error": c.Errors[0].Error(),
			})
		}
	}
}

// CORSMiddleware 跨域支持
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
