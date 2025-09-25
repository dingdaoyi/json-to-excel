package config

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"json-to-excel/internal"
	"net/http"
)

func McpHandler(host, port string) http.Handler {

	// MCP Server
	mcpServer := mcp.NewServer(&mcp.Implementation{Name: "JsonToExcel",
		Version: "1.0.0",
	}, nil)
	svc := internal.NewExcelService(internal.Config{
		Host: host,
		Port: port,
	})
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
	return handler
}
