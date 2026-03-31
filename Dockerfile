# 多阶段构建

# 阶段 1: 构建阶段
FROM golang:1.25-alpine AS builder

# 安装必要的工具
RUN apk add --no-cache git ca-certificates tzdata

# 设置工作目录
WORKDIR /app

# 复制 go mod 文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /app/bin/jetwash-platform \
    cmd/server/main.go

# 阶段 2: 运行阶段
FROM alpine:latest

# 安装 ca-certificates（用于 HTTPS 请求）
RUN apk --no-cache add ca-certificates tzdata

# 创建非 root 用户
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

# 设置工作目录
WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/bin/jetwash-platform /app/bin/jetwash-platform

# 复制配置文件（可选，也可以通过 volume 挂载）
COPY config.yaml /app/config.yaml

# 更改文件所有者
RUN chown -R appuser:appuser /app

# 切换到非 root 用户
USER appuser

# 暴露端口
EXPOSE 8080

# 设置环境变量
ENV GIN_MODE=release

# 运行应用
CMD ["/app/bin/jetwash-platform", "-config", "/app/config.yaml"]
