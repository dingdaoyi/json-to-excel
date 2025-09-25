package internal

import (
	"context"
	"fmt"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/xuri/excelize/v2"
	"log"
	"os"
	"path/filepath"
)

var (
	downloadDir = "./downloads"
)

const (
	ErrNoData    = "未提供需要转换的数据"
	ErrNoHeaders = "未提供表头信息"
)

type Config struct {
	Host string
	Port string
}

// JSONToExcelParam 定义 JSON 转 Excel 的请求参数。
type JSONToExcelParam struct {
	// Headers 定义 Excel 表格的表头映射关系，key 为 JSON 字段名，value 为表头显示名。
	Headers map[string]string `json:"headers"`
	// Data 包含需要转换的数据记录。
	Data []map[string]any `json:"data"`
}

type ExcelService struct {
	Host string
	Port string
}

func NewExcelService(cfg Config) *ExcelService {
	return &ExcelService{
		Host: cfg.Host,
		Port: cfg.Port,
	}
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
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		return nil, nil, err
	}

	// 写入临时文件
	tmpFile, err := os.CreateTemp(downloadDir, "result-*.xlsx")
	if err != nil {
		return nil, nil, err
	}
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
