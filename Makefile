.PHONY: all build run clean test deps

# 变量定义
APP_NAME=jetwash
CMD_DIR=cmd/server
BUILD_DIR=bin
CONFIG_FILE=config.yaml

# 默认目标
all: deps build

# 安装依赖
deps:
	go mod download
	go mod tidy

# 构建
build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(APP_NAME) $(CMD_DIR)/main.go
	@echo "Build complete: $(BUILD_DIR)/$(APP_NAME)"

# 运行
run:
	@echo "Running $(APP_NAME)..."
	go run $(CMD_DIR)/main.go -config $(CONFIG_FILE)

# 运行（使用配置文件）
run-dev:
	@echo "Running $(APP_NAME) in dev mode..."
	go run $(CMD_DIR)/main.go -config $(CONFIG_FILE)

# 测试
test:
	@echo "Running tests..."
	go test -v ./...

# 测试（带覆盖率）
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# 清理
clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	@echo "Clean complete"

# 格式化代码
fmt:
	@echo "Formatting code..."
	go fmt ./...

# 代码检查
lint:
	@echo "Running linter..."
	golangci-lint run

# 数据库迁移
migrate-up:
	@echo "Running database migrations..."
	go run $(CMD_DIR)/main.go -config $(CONFIG_FILE) migrate up

# 数据库回滚
migrate-down:
	@echo "Rolling back database migrations..."
	go run $(CMD_DIR)/main.go -config $(CONFIG_FILE) migrate down

# Docker 构建
docker-build:
	@echo "Building Docker image..."
	docker build -t $(APP_NAME):latest .

# Docker 运行
docker-run:
	@echo "Running Docker container..."
	docker run -p 8080:8080 --env-file .env $(APP_NAME):latest

# 帮助
help:
	@echo "Available targets:"
	@echo "  all           - Install dependencies and build (default)"
	@echo "  deps          - Install dependencies"
	@echo "  build         - Build the application"
	@echo "  run           - Run the application"
	@echo "  run-dev       - Run the application in dev mode"
	@echo "  test          - Run tests"
	@echo "  test-coverage - Run tests with coverage"
	@echo "  clean         - Clean build artifacts"
	@echo "  fmt           - Format code"
	@echo "  lint          - Run linter"
	@echo "  migrate-up    - Run database migrations"
	@echo "  migrate-down  - Rollback database migrations"
	@echo "  docker-build  - Build Docker image"
	@echo "  docker-run    - Run Docker container"
	@echo "  help          - Show this help message"
