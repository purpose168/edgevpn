# ============================================================================
# EdgeVPN Makefile
# 用途：项目构建、测试、部署自动化
# 作者：purpose168@outlook.com
# ============================================================================

# ============================================================================
# 变量定义
# ============================================================================

# 项目信息
PROJECT_NAME := edgevpn
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "v0.0.0-dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')

# Go 相关变量
GO := go
GO_VERSION := 1.25
GOCMD := $(GO)
GOBUILD := $(GO) build
GOCLEAN := $(GO) clean
GOTEST := $(GO) test
GOGET := $(GO) get
GOMOD := $(GO) mod

# 构建标志
LDFLAGS := -s -w \
	-X main.Version=$(VERSION) \
	-X main.Commit=$(COMMIT) \
	-X main.BuildTime=$(BUILD_TIME)

# 静态编译标志
CGO_ENABLED := 0

# 输出目录
DIST_DIR := dist
BIN_DIR := bin

# 二进制文件
BINARY := $(PROJECT_NAME)
BINARY_LINUX := $(BIN_DIR)/$(PROJECT_NAME)-linux-amd64
BINARY_DARWIN := $(BIN_DIR)/$(PROJECT_NAME)-darwin-amd64
BINARY_WINDOWS := $(BIN_DIR)/$(PROJECT_NAME)-windows-amd64.exe

# Docker 相关
DOCKER := docker
DOCKER_IMAGE := $(PROJECT_NAME)
DOCKER_TAG := latest
DOCKER_REGISTRY := quay.io/purpose168

# 颜色输出
GREEN := \033[0;32m
YELLOW := \033[0;33m
RED := \033[0;31m
NC := \033[0m

# ============================================================================
# 默认目标
# ============================================================================

.DEFAULT_GOAL := help

# ============================================================================
# 构建目标
# ============================================================================

## build: 构建当前平台的二进制文件
.PHONY: build
build: | $(BIN_DIR)
	@echo "$(GREEN)正在构建 $(PROJECT_NAME)...$(NC)"
	$(GOBUILD) -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY) .
	@echo "$(GREEN)构建完成: $(BIN_DIR)/$(BINARY)$(NC)"

## build-static: 静态编译（无 CGO 依赖）
.PHONY: build-static
build-static: | $(BIN_DIR)
	@echo "$(GREEN)正在静态编译 $(PROJECT_NAME)...$(NC)"
	CGO_ENABLED=$(CGO_ENABLED) $(GOBUILD) -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY) .
	@echo "$(GREEN)静态编译完成: $(BIN_DIR)/$(BINARY)$(NC)"

## build-all: 构建所有平台的二进制文件（使用 GoReleaser）
.PHONY: build-all
build-all:
	@echo "$(GREEN)正在构建所有平台版本...$(NC)"
	goreleaser build --clean --snapshot
	@echo "$(GREEN)所有平台构建完成$(NC)"

## build-linux: 构建 Linux amd64 版本
.PHONY: build-linux
build-linux: | $(BIN_DIR)
	@echo "$(GREEN)正在构建 Linux amd64 版本...$(NC)"
	GOOS=linux GOARCH=amd64 CGO_ENABLED=$(CGO_ENABLED) \
		$(GOBUILD) -ldflags="$(LDFLAGS)" -o $(BINARY_LINUX) .
	@echo "$(GREEN)构建完成: $(BINARY_LINUX)$(NC)"

## build-darwin: 构建 macOS amd64 版本
.PHONY: build-darwin
build-darwin: | $(BIN_DIR)
	@echo "$(GREEN)正在构建 macOS amd64 版本...$(NC)"
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=$(CGO_ENABLED) \
		$(GOBUILD) -ldflags="$(LDFLAGS)" -o $(BINARY_DARWIN) .
	@echo "$(GREEN)构建完成: $(BINARY_DARWIN)$(NC)"

## build-windows: 构建 Windows amd64 版本
.PHONY: build-windows
build-windows: | $(BIN_DIR)
	@echo "$(GREEN)正在构建 Windows amd64 版本...$(NC)"
	GOOS=windows GOARCH=amd64 CGO_ENABLED=$(CGO_ENABLED) \
		$(GOBUILD) -ldflags="$(LDFLAGS)" -o $(BINARY_WINDOWS) .
	@echo "$(GREEN)构建完成: $(BINARY_WINDOWS)$(NC)"

## release: 发布版本（使用 GoReleaser）
.PHONY: release
release:
	@echo "$(GREEN)正在发布版本...$(NC)"
	goreleaser release --clean
	@echo "$(GREEN)发布完成$(NC)"

# ============================================================================
# 测试目标
# ============================================================================

## test: 运行所有测试
.PHONY: test
test:
	@echo "$(GREEN)正在运行测试...$(NC)"
	$(GOTEST) -v -race -coverprofile=coverage.out ./...
	@echo "$(GREEN)测试完成$(NC)"

## test-coverage: 运行测试并生成覆盖率报告
.PHONY: test-coverage
test-coverage: test
	@echo "$(GREEN)正在生成覆盖率报告...$(NC)"
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)覆盖率报告已生成: coverage.html$(NC)"

## test-race: 运行竞态检测
.PHONY: test-race
test-race:
	@echo "$(GREEN)正在运行竞态检测...$(NC)"
	$(GOTEST) -race ./...
	@echo "$(GREEN)竞态检测完成$(NC)"

## benchmark: 运行性能测试
.PHONY: benchmark
benchmark:
	@echo "$(GREEN)正在运行性能测试...$(NC)"
	$(GOTEST) -bench=. -benchmem ./...
	@echo "$(GREEN)性能测试完成$(NC)"

# ============================================================================
# 依赖管理
# ============================================================================

## deps: 下载项目依赖
.PHONY: deps
deps:
	@echo "$(GREEN)正在下载依赖...$(NC)"
	$(GOMOD) download
	@echo "$(GREEN)依赖下载完成$(NC)"

## deps-verify: 验证依赖完整性
.PHONY: deps-verify
deps-verify:
	@echo "$(GREEN)正在验证依赖...$(NC)"
	$(GOMOD) verify
	@echo "$(GREEN)依赖验证完成$(NC)"

## deps-update: 更新所有依赖
.PHONY: deps-update
deps-update:
	@echo "$(GREEN)正在更新依赖...$(NC)"
	$(GOMOD) tidy
	$(GOGET) -u ./...
	@echo "$(GREEN)依赖更新完成$(NC)"

## deps-clean: 清理依赖缓存
.PHONY: deps-clean
deps-clean:
	@echo "$(YELLOW)正在清理依赖缓存...$(NC)"
	$(GOCLEAN) -modcache
	@echo "$(GREEN)依赖缓存清理完成$(NC)"

# ============================================================================
# Docker 目标
# ============================================================================

## docker-build: 构建 Docker 镜像
.PHONY: docker-build
docker-build:
	@echo "$(GREEN)正在构建 Docker 镜像...$(NC)"
	$(DOCKER) build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	@echo "$(GREEN)Docker 镜像构建完成: $(DOCKER_IMAGE):$(DOCKER_TAG)$(NC)"

## docker-build-version: 构建带版本标签的 Docker 镜像
.PHONY: docker-build-version
docker-build-version:
	@echo "$(GREEN)正在构建 Docker 镜像 $(VERSION)...$(NC)"
	$(DOCKER) build -t $(DOCKER_IMAGE):$(VERSION) .
	@echo "$(GREEN)Docker 镜像构建完成: $(DOCKER_IMAGE):$(VERSION)$(NC)"

## docker-push: 推送 Docker 镜像到仓库
.PHONY: docker-push
docker-push:
	@echo "$(GREEN)正在推送 Docker 镜像...$(NC)"
	$(DOCKER) tag $(DOCKER_IMAGE):$(DOCKER_TAG) $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(DOCKER_TAG)
	$(DOCKER) push $(DOCKER_REGISTRY)/$(DOCKER_IMAGE):$(DOCKER_TAG)
	@echo "$(GREEN)Docker 镜像推送完成$(NC)"

## docker-run: 使用 Docker 运行 EdgeVPN
.PHONY: docker-run
docker-run:
	@echo "$(GREEN)正在启动 Docker 容器...$(NC)"
	$(DOCKER) run -d --name $(PROJECT_NAME) \
		--network host \
		--cap-add NET_ADMIN \
		--device /dev/net/tun:/dev/net/tun \
		-e EDGEVPNTOKEN=$(EDGEVPNTOKEN) \
		$(DOCKER_IMAGE):$(DOCKER_TAG)
	@echo "$(GREEN)容器已启动$(NC)"

## docker-stop: 停止 Docker 容器
.PHONY: docker-stop
docker-stop:
	@echo "$(YELLOW)正在停止 Docker 容器...$(NC)"
	$(DOCKER) stop $(PROJECT_NAME) || true
	$(DOCKER) rm $(PROJECT_NAME) || true
	@echo "$(GREEN)容器已停止$(NC)"

## docker-compose-up: 使用 Docker Compose 启动服务
.PHONY: docker-compose-up
docker-compose-up:
	@echo "$(GREEN)正在启动 Docker Compose 服务...$(NC)"
	docker-compose up -d
	@echo "$(GREEN)服务已启动$(NC)"

## docker-compose-down: 停止 Docker Compose 服务
.PHONY: docker-compose-down
docker-compose-down:
	@echo "$(YELLOW)正在停止 Docker Compose 服务...$(NC)"
	docker-compose down
	@echo "$(GREEN)服务已停止$(NC)"

## docker-compose-logs: 查看 Docker Compose 日志
.PHONY: docker-compose-logs
docker-compose-logs:
	docker-compose logs -f

# ============================================================================
# 代码质量
# ============================================================================

## lint: 运行代码检查
.PHONY: lint
lint:
	@echo "$(GREEN)正在运行代码检查...$(NC)"
	@which golangci-lint > /dev/null || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	golangci-lint run ./...
	@echo "$(GREEN)代码检查完成$(NC)"

## fmt: 格式化代码
.PHONY: fmt
fmt:
	@echo "$(GREEN)正在格式化代码...$(NC)"
	gofmt -s -w .
	@echo "$(GREEN)代码格式化完成$(NC)"

## vet: 运行 go vet
.PHONY: vet
vet:
	@echo "$(GREEN)正在运行 go vet...$(NC)"
	$(GO) vet ./...
	@echo "$(GREEN)go vet 完成$(NC)"

# ============================================================================
# 配置和安装
# ============================================================================

## config: 生成默认配置文件
.PHONY: config
config:
	@echo "$(GREEN)正在生成配置文件...$(NC)"
	@if [ -f $(BIN_DIR)/$(BINARY) ]; then \
		$(BIN_DIR)/$(BINARY) -g > config.yaml; \
	else \
		echo "$(RED)请先运行 make build 构建二进制文件$(NC)"; \
		exit 1; \
	fi
	@echo "$(GREEN)配置文件已生成: config.yaml$(NC)"

## install: 安装到系统路径
.PHONY: install
install: build
	@echo "$(GREEN)正在安装到系统路径...$(NC)"
	sudo cp $(BIN_DIR)/$(BINARY) /usr/local/bin/$(PROJECT_NAME)
	sudo chmod +x /usr/local/bin/$(PROJECT_NAME)
	@echo "$(GREEN)安装完成: /usr/local/bin/$(PROJECT_NAME)$(NC)"

## uninstall: 从系统路径卸载
.PHONY: uninstall
uninstall:
	@echo "$(YELLOW)正在卸载...$(NC)"
	sudo rm -f /usr/local/bin/$(PROJECT_NAME)
	@echo "$(GREEN)卸载完成$(NC)"

# ============================================================================
# 清理目标
# ============================================================================

## clean: 清理构建产物
.PHONY: clean
clean:
	@echo "$(YELLOW)正在清理构建产物...$(NC)"
	$(GOCLEAN)
	rm -rf $(BIN_DIR)
	rm -rf $(DIST_DIR)
	rm -f coverage.out coverage.html
	@echo "$(GREEN)清理完成$(NC)"

# ============================================================================
# 开发工具
# ============================================================================

## dev: 开发模式运行
.PHONY: dev
dev: build
	@echo "$(GREEN)正在启动开发模式...$(NC)"
	./$(BIN_DIR)/$(BINARY) --debug --log-level debug

## run: 运行程序
.PHONY: run
run: build
	./$(BIN_DIR)/$(BINARY)

## api: 启动 API 服务
.PHONY: api
api: build
	@echo "$(GREEN)正在启动 API 服务...$(NC)"
	./$(BIN_DIR)/$(BINARY) --api --api-listen 127.0.0.1:8080

# ============================================================================
# 帮助
# ============================================================================

## help: 显示帮助信息
.PHONY: help
help:
	@echo ""
	@echo "$(GREEN)EdgeVPN Makefile 帮助$(NC)"
	@echo ""
	@echo "$(YELLOW)使用方法:$(NC)"
	@echo "  make [目标]"
	@echo ""
	@echo "$(YELLOW)构建目标:$(NC)"
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | grep -E '^(build|release)' | column -t -s ':'
	@echo ""
	@echo "$(YELLOW)测试目标:$(NC)"
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | grep -E '^(test|benchmark)' | column -t -s ':'
	@echo ""
	@echo "$(YELLOW)Docker 目标:$(NC)"
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | grep -E '^docker' | column -t -s ':'
	@echo ""
	@echo "$(YELLOW)依赖管理:$(NC)"
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | grep -E '^deps' | column -t -s ':'
	@echo ""
	@echo "$(YELLOW)代码质量:$(NC)"
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | grep -E '^(lint|fmt|vet)' | column -t -s ':'
	@echo ""
	@echo "$(YELLOW)其他目标:$(NC)"
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | grep -vE '^(build|release|test|benchmark|docker|deps|lint|fmt|vet)' | column -t -s ':'
	@echo ""

# ============================================================================
# 目录创建
# ============================================================================

$(BIN_DIR):
	@mkdir -p $(BIN_DIR)

$(DIST_DIR):
	@mkdir -p $(DIST_DIR)
