package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	"json-to-excel/config"
	"json-to-excel/internal"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	host        = flag.String("host", "localhost", "监听地址")
	port        = flag.String("port", "8080", "监听端口")
	downloadDir = "./downloads"
)

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
		ExpirationDur: 2 * time.Minute,
		CleanupTick:   30 * time.Second,
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

	log.Printf("服务启动 %s", addr)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("服务启动失败: %v", err)
		}
	}()
	// 等待中断信号
	<-sigChan
	log.Println("正在关闭服务...")

	// 然后关闭 Excel 服务
	if err := mcpHandler.Close(); err != nil {
		log.Printf("Excel service Close: %v", err)
	}

	log.Println("服务已完全关闭")
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
