package internal

import (
	"context"
	"fmt"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/xuri/excelize/v2"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	ErrNoData    = "未提供需要转换的数据"
	ErrNoHeaders = "未提供表头信息"
)

type Config struct {
	Host          string
	Port          string
	TempDir       string
	ExpirationDur time.Duration // 文件过期时间
	CleanupTick   time.Duration // 清理检查间隔
}

// JSONToExcelParam 定义 JSON 转 Excel 的请求参数。
type JSONToExcelParam struct {
	// Headers 定义 Excel 表格的表头映射关系，key 为 JSON 字段名，value 为表头显示名。
	Headers map[string]string `json:"headers"`
	// Data 包含需要转换的数据记录。
	Data []map[string]any `json:"data"`
}

type ExcelService struct {
	Host          string
	Port          string
	TempDir       string               // 临时文件目录
	excelFiles    map[string]time.Time // 文件名到过期时间的映射
	mutex         sync.RWMutex         // 保护 ExcelFiles 的并发访问
	cleanupTick   time.Duration        // 清理间隔
	expirationDur time.Duration        // 文件过期时间
}

func NewExcelService(cfg Config) (*ExcelService, error) {
	// 创建临时目录
	if cfg.TempDir == "" {
		cfg.TempDir = filepath.Join(os.TempDir(), "excel-tmp")
	}
	if err := os.MkdirAll(cfg.TempDir, 0755); err != nil {
		return nil, fmt.Errorf("创建临时目录失败: %w", err)
	}

	s := &ExcelService{
		Host:          cfg.Host,
		Port:          cfg.Port,
		TempDir:       cfg.TempDir,
		excelFiles:    make(map[string]time.Time),
		expirationDur: cfg.ExpirationDur,
		cleanupTick:   cfg.CleanupTick,
	}

	// 启动清理任务
	go s.startCleanupTask()

	return s, nil
}

// 创建Excel文件并记录过期时间
func (s *ExcelService) addCleanFile(filename string) {
	s.mutex.Lock()
	t := time.Now().Add(s.expirationDur)
	log.Printf("添加文件时间 %s: %v", filename, t)
	s.excelFiles[filename] = t
	s.mutex.Unlock()

}

// 清理过期文件的定时任务
func (s *ExcelService) startCleanupTask() {
	ticker := time.NewTicker(s.cleanupTick)
	defer ticker.Stop()

	for range ticker.C {
		s.cleanupExpiredFiles()
	}
}

// 清理过期文件
func (s *ExcelService) cleanupExpiredFiles() {
	now := time.Now()
	log.Print("清理临时文件开始:\n")
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for filePath, expireTime := range s.excelFiles {
		log.Printf("loginpath:%s,expireTime:%s", filePath, expireTime)
		if now.After(expireTime) {
			if err := os.Remove(filePath); err != nil {
				log.Printf("删除过期文件失败 %s: %v", filePath, err)
			} else {
				log.Printf("已删除过期文件: %s", filePath)
				delete(s.excelFiles, filePath)
			}
		}
	}
}

// Shutdown 服务停止时清理所有文件
func (s *ExcelService) Shutdown() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 删除所有文件
	for filePath := range s.excelFiles {
		if err := os.Remove(filePath); err != nil {
			log.Printf("关闭时删除文件失败 %s: %v", filePath, err)
		}
	}

	// 清空映射
	s.excelFiles = make(map[string]time.Time)

	// 删除临时目录
	return os.RemoveAll(s.TempDir)
}

// validateParams 验证输入参数的有效性。
func (s *ExcelService) validateParams(args JSONToExcelParam) error {
	if len(args.Data) == 0 {
		return fmt.Errorf(ErrNoData)
	}
	if len(args.Headers) == 0 {
		return fmt.Errorf(ErrNoHeaders)
	}
	return nil
}
func (s *ExcelService) JsonToExcel(ctx context.Context, req *mcp.CallToolRequest, args JSONToExcelParam) (*mcp.CallToolResult, any, error) {

	if err := s.validateParams(args); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: err.Error(),
				},
			},
		}, nil, nil
	}
	f := excelize.NewFile()
	sheet := "Sheet1"

	// 确保有默认的 Sheet1
	index, er := f.GetSheetIndex(sheet)
	if er != nil {
		return nil, nil, er
	}

	if index == -1 {
		f.NewSheet(sheet)
	}
	// 写表头：headers 的 value 是表头中文
	colIndex := 1
	cols := make([]string, 0, len(args.Headers))
	for field, title := range args.Headers {
		cell, _ := excelize.CoordinatesToCellName(colIndex, 1)
		f.SetCellValue(sheet, cell, title)
		cols = append(cols, field) // 保存字段顺序
		colIndex++
	}

	// 写数据：从 data 里取 field 的值
	for i, row := range args.Data {
		rowIndex := i + 2 // 从第2行开始
		for j, field := range cols {
			cell, _ := excelize.CoordinatesToCellName(j+1, rowIndex)
			f.SetCellValue(sheet, cell, row[field])
		}
	}

	// 确保 downloads 目录存在
	if err := os.MkdirAll(s.TempDir, 0755); err != nil {
		return nil, nil, err
	}

	// 写入临时文件
	tmpFile, err := os.CreateTemp(s.TempDir, "result-*.xlsx")
	if err != nil {
		return nil, nil, err
	}
	s.addCleanFile(tmpFile.Name())
	defer tmpFile.Close()

	if err := f.Write(tmpFile); err != nil {
		return nil, nil, err
	}

	// 构造下载链接
	fileName := filepath.Base(tmpFile.Name())
	fileURL := fmt.Sprintf("http://%s:%s/downloads/%s", s.Host, s.Port, fileName)
	log.Printf("请求生成excel:%s", fileURL)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.ResourceLink{
				URI:      fileURL,
				Name:     "json转换结果",
				Title:    "data",
				MIMEType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
			},
		},
	}, nil, nil
}
