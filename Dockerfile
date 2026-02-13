# 定义链接器标志参数
ARG LDFLAGS=-s -w

# 使用基于 Golang 1.20-alpine 的临时构建镜像
FROM golang:1.25-alpine as builder

# 设置环境变量：链接器标志并禁用 CGO
ENV LDFLAGS=$LDFLAGS CGO_ENABLED=0

# 将当前目录内容添加到容器中的工作目录
ADD . /work

# 设置容器内的当前工作目录
WORKDIR /work

# 安装 git 并使用提供的链接器标志构建 edgevpn 二进制文件
# --no-cache 标志确保包缓存不存储在层中，从而减小镜像大小
RUN apk add --no-cache git && \
    go build -ldflags="$LDFLAGS" -o edgevpn

# TODO: 移动到 distroless

# 使用新的、干净的 alpine 镜像作为最终阶段
FROM alpine

# 将 edgevpn 二进制文件从构建阶段复制到最终镜像
COPY --from=builder /work/edgevpn /usr/bin/edgevpn

# 定义容器启动时将运行的命令
ENTRYPOINT ["/usr/bin/edgevpn"]
