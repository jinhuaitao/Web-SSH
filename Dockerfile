FROM golang:1.21 AS builder

WORKDIR /app

# 复制源代码和模块文件
COPY main.go go.mod ./

# 初始化模块并下载依赖
RUN go mod tidy && \
    go mod download

# 编译应用
RUN CGO_ENABLED=0 GOOS=linux go build -o webssh .

# 使用轻量级的alpine镜像
FROM alpine:latest

WORKDIR /app

# 从builder阶段复制编译好的应用
COPY --from=builder /app/webssh .

# 复制静态文件和模板
COPY static/ ./static/
COPY templates/ ./templates/

# 暴露端口
EXPOSE 8080

# 运行应用
CMD ["./webssh"]