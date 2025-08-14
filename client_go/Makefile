# HotKey Go Client Makefile

.PHONY: all build test clean deps fmt vet lint run-example run-advanced help

# 默认目标
all: deps fmt vet test build

# 安装依赖
deps:
	@echo "Installing dependencies..."
	go mod tidy
	go mod download

# 格式化代码
fmt:
	@echo "Formatting code..."
	go fmt ./...

# 代码检查
vet:
	@echo "Running go vet..."
	go vet ./...

# 运行测试
test:
	@echo "Running tests..."
	go test -v ./...

# 运行基准测试
bench:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

# 运行测试并生成覆盖率报告
test-coverage:
	@echo "Running tests with coverage..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# 构建
build:
	@echo "Building..."
	go build -v ./...

# 构建示例程序
build-examples:
	@echo "Building examples..."
	go build -o bin/basic_usage ./examples/basic_usage.go
	go build -o bin/advanced_usage ./examples/advanced_usage.go
	go build -o bin/enhanced_usage ./examples/enhanced_usage.go

# 运行基本示例
run-example: build-examples
	@echo "Running basic example..."
	./bin/basic_usage

# 运行高级示例
run-advanced: build-examples
	@echo "Running advanced example..."
	./bin/advanced_usage

# 运行增强示例
run-enhanced: build-examples
	@echo "Running enhanced example..."
	./bin/enhanced_usage

# 代码质量检查（需要安装golangci-lint）
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# 清理
clean:
	@echo "Cleaning..."
	go clean
	rm -rf bin/
	rm -f coverage.out coverage.html

# 初始化项目（首次使用）
init:
	@echo "Initializing project..."
	go mod init github.com/jd/platform/hotkey/client-go || true
	$(MAKE) deps

# 更新依赖
update:
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy

# 检查模块
mod-check:
	@echo "Checking modules..."
	go mod verify
	go mod tidy -diff

# 生成文档
doc:
	@echo "Generating documentation..."
	@if command -v godoc >/dev/null 2>&1; then \
		echo "Starting godoc server on http://localhost:6060"; \
		godoc -http=:6060; \
	else \
		echo "godoc not installed. Install with: go install golang.org/x/tools/cmd/godoc@latest"; \
	fi

# 安装开发工具
install-tools:
	@echo "Installing development tools..."
	go install golang.org/x/tools/cmd/godoc@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# 运行所有检查
check: fmt vet lint test

# 发布准备
release-check: clean check test-coverage
	@echo "Release checks completed successfully!"

# Docker构建（如果需要）
docker-build:
	@echo "Building Docker image..."
	docker build -t hotkey-go-client .

# 显示帮助
help:
	@echo "Available targets:"
	@echo "  all           - Run deps, fmt, vet, test, and build"
	@echo "  deps          - Install dependencies"
	@echo "  fmt           - Format code"
	@echo "  vet           - Run go vet"
	@echo "  lint          - Run linter (requires golangci-lint)"
	@echo "  test          - Run tests"
	@echo "  bench         - Run benchmarks"
	@echo "  test-coverage - Run tests with coverage report"
	@echo "  build         - Build the project"
	@echo "  build-examples- Build example programs"
	@echo "  run-example   - Run basic example"
	@echo "  run-advanced  - Run advanced example"
	@echo "  run-enhanced  - Run enhanced example"
	@echo "  clean         - Clean build artifacts"
	@echo "  init          - Initialize project (first time setup)"
	@echo "  update        - Update dependencies"
	@echo "  mod-check     - Check and verify modules"
	@echo "  doc           - Start documentation server"
	@echo "  install-tools - Install development tools"
	@echo "  check         - Run all checks (fmt, vet, lint, test)"
	@echo "  release-check - Run all release preparation checks"
	@echo "  docker-build  - Build Docker image"
	@echo "  help          - Show this help message"

# 设置默认shell
SHELL := /bin/bash
