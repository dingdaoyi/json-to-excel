
FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY . .

# 设置 Go 环境变量
ENV GO111MODULE=on \
GOPROXY=https://goproxy.cn,direct

# 构建应用
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o json-to-excel

FROM alpine:latest

WORKDIR /app

# 从 builder 阶段复制二进制文件
COPY --from=builder /app/json-to-excel .

# 创建临时文件目录
RUN mkdir -p /app/downloads

# 设置时区
RUN apk --no-cache add tzdata && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone && \
    apk del tzdata

# 环境变量配置
ENV HOST=0.0.0.0 \
    PORT=8080 \
    LOG_LEVEL=info \
    DOWNLOAD_DIR=/app/downloads \
    FILE_EXPIRATION=2m \
    CLEANUP_INTERVAL=30s

EXPOSE 8080

ENTRYPOINT ["./json-to-excel"]