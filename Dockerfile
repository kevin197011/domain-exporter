# 构建阶段
FROM golang:1.24-alpine AS builder

WORKDIR /app

# 复制go mod文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY *.go ./

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o domain-exporter .

# 运行阶段
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

# 创建应用用户和目录
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

# 创建Nacos所需的目录并设置权限
RUN mkdir -p /app/logs/nacos /app/cache/nacos && \
    chown -R appuser:appgroup /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/domain-exporter .
RUN chown appuser:appgroup domain-exporter

# 切换到非root用户
USER appuser

# 暴露端口
EXPOSE 8080

# 运行应用
CMD ["./domain-exporter"]