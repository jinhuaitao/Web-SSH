FROM golang:1.21 AS builder

# 完全启用IPv6支持
RUN echo "net.ipv6.conf.all.disable_ipv6 = 0" >> /etc/sysctl.conf && \
    echo "net.ipv6.conf.default.disable_ipv6 = 0" >> /etc/sysctl.conf && \
    echo "net.ipv6.conf.lo.disable_ipv6 = 0" >> /etc/sysctl.conf && \
    sysctl -p

# 添加hosts配置确保host.docker.internal可用
RUN echo "127.0.0.1 host.docker.internal" >> /etc/hosts
