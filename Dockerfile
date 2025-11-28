# --- 阶段 1：构建阶段 ---
# 使用官方的 Golang 镜像作为基础。这个镜像包含了完整的 Go 编译环境。
# 我们给这个阶段起个名字叫 "builder"。
FROM golang:1.25-alpine AS builder

# 关键配置：启用模块模式 + 配置国内代理
ENV GO111MODULE=on
ENV GOPROXY=https://goproxy.cn,direct

# 在容器内创建一个工作目录。
WORKDIR /app

# 将 go.mod 和 go.sum 文件复制到容器中。
# 这一步很重要，因为 Docker 会缓存这一层。只要这两个文件不变，就不需要重新下载依赖。
COPY go.mod go.sum ./

# 下载项目所需的所有依赖。
RUN go mod download

# 将当前目录下的所有源代码复制到容器中。
COPY . .

# 编译 Go 应用。
# CGO_ENABLED=0: 禁用 CGO，生成一个静态链接的二进制文件，不依赖系统库。
# GOOS=linux: 指定目标操作系统为 Linux。
# -o app: 将编译后的可执行文件命名为 "app"。
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app .

# --- 阶段 2：运行阶段 ---
# 使用一个非常轻量级的 Alpine Linux 镜像作为最终的运行环境。
# 这个镜像比构建阶段的 Golang 镜像小得多（只有几 MB）。
FROM alpine:3.18

# 在容器内创建一个非 root 用户，并切换到该用户，以提高安全性。
RUN adduser -D -H appuser
USER appuser

# 在容器内创建一个工作目录。
WORKDIR /home/appuser

# 从 "builder" 阶段将编译好的二进制文件 "app" 复制到当前镜像中。
# 这是多阶段构建的核心：只复制最终需要的产物，而不是整个构建环境。
COPY --from=builder /app/app .

# 声明容器将对外暴露 8080 端口。
# 这只是一个元数据声明，告诉使用者这个容器需要映射哪个端口。
EXPOSE 8080

# 定义容器启动时要执行的命令。
# 这里就是运行我们编译好的 "app" 程序。
CMD ["./app"]