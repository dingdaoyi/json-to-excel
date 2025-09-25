package config

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"json-to-excel/internal"
	"net/http"
)

type MCPHandler struct {
	handler http.Handler
	svc     *internal.ExcelService
}

func NewMCPHandler(config internal.Config) *MCPHandler {
	// 创建 Excel 服务
	svc, _ := internal.NewExcelService(config)

	// MCP Server
	mcpServer := mcp.NewServer(&mcp.Implementation{
		Name:    "JsonToExcel",
		Version: "1.0.0",
	}, nil)

	mcp.AddTool(mcpServer, &mcp.Tool{Name: "jsonToExcel",
		Description: `将结构化的 JSON 数据转换为 Excel 文件（.xlsx），并返回下载链接。
							适用于把 API 数据、表格类 JSON 转换为可下载的 Excel 格式。
							参数格式:
							{
							  "headers": {
								"字段名": "表头显示名"
							  },
							  "data": [
								{
								  "字段名1": "值1",
								  "字段名2": "值2"
								}
							  ]
							}
							返回结果:
							- ResourceLink: Excel 文件下载链接（.xlsx 格式）`,
	}, svc.JsonToExcel)

	handler := mcp.NewStreamableHTTPHandler(func(request *http.Request) *mcp.Server {
		return mcpServer
	}, &mcp.StreamableHTTPOptions{})

	return &MCPHandler{
		handler: handler,
		svc:     svc,
	}
}

func (h *MCPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.handler.ServeHTTP(w, r)
}

// Close 提供关闭服务的方法
func (h *MCPHandler) Close() error {
	if h.svc != nil {
		return h.svc.Shutdown()
	}
	return nil
}
