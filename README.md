## JSON to Excel MCP Service
一个基于 MCP 协议的 JSON 转 Excel 服务，支持将结构化的 JSON 数据转换为带自定义表头的 Excel 文件。

### 功能介绍
将 JSON 数据转换为 Excel 文件，支持：

自定义表头映射

返回文件下载链接

文件自动清理
#  通过 Docker 运行
```shell
docker build -t json-to-excel:latest .
# 使用docker-compose运行（推荐）
docker-compose up -d

# 或直接运行容器
docker run -d -p 8080:8080 \
  -e HOST=0.0.0.0 \
  -e PORT=8080 \
  -e BASE_URL=http://localhost:8080 \
  -e FILE_EXPIRATION=5m \
  -v ./downloads:/app/downloads \
  json-to-excel:latest
```
### 直接运行
#### 编译
```shell
go build -o json-to-excel
```

#### 使用环境变量运行
```shell
# 复制环境变量配置文件
cp .env.example .env

# 编辑配置
vim .env

# 加载环境变量并运行
source .env && ./json-to-excel
```

#### 使用命令行参数运行
```shell
./json-to-excel -port 8080 -host localhost
```

### API 调用示例
```json
{
  "headers": {
    "name": "姓名",
    "age": "年龄",
    "email": "邮箱"
  },
  "data": [
    {
      "name": "张三",
      "age": 25,
      "email": "zhangsan@example.com"
    }
  ]
}
```
### 配置说明

#### 环境变量配置（推荐）
| 环境变量 | 默认值 | 说明 |
|---------|--------|------|
| HOST | localhost | 服务监听地址 |
| PORT | 8080 | 服务端口号 |
| BASE_URL | - | 外部访问基础URL（Docker部署时必需） |
| LOG_LEVEL | info | 日志级别（debug/info/warn/error） |
| DOWNLOAD_DIR | ./downloads | 文件下载目录 |
| FILE_EXPIRATION | 2m | 文件过期时间 |
| CLEANUP_INTERVAL | 30s | 清理检查间隔 |

#### 命令行参数
-port: 服务端口号（默认：8080）
-host: 服务地址（默认：localhost）

注：环境变量优先级高于命令行参数
