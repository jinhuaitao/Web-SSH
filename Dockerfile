FROM golang:1.21 AS builder

# 安装必要工具和启用IPv6
RUN apt-get update && apt-get install -y \
    net-tools \
    iputils-ping && \
    echo "net.ipv6.conf.all.disable_ipv6 = 0" >> /etc/sysctl.conf && \
    echo "net.ipv6.conf.default.disable_ipv6 = 0" >> /etc/sysctl.conf && \
    echo "net.ipv6.conf.lo.disable_ipv6 = 0" >> /etc/sysctl.conf && \
    sysctl -p

# 添加hosts配置
RUN echo "127.0.0.1 host.docker.internal" >> /etc/hosts

WORKDIR /app
COPY go.mod .
RUN go mod download
COPY . .

# 编译应用
RUN CGO_ENABLED=0 GOOS=linux go build -o webssh .

# 生产阶段
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /app/webssh .
COPY static/ ./static/
COPY templates/ ./templates/

EXPOSE 8080
CMD ["./webssh"]
