.PHONY: build build-grad build-gractl clean test generate help minikube-start minikube-stop minikube-status dev dev-stop dev-debug

# Build configuration
OUT_DIR=out
GO_FILES=$(shell find . -name "*.go" -type f)

# Default target
default: help

# Build both binaries
build: build-grad build-gractl

# Build grad binary
build-grad: $(OUT_DIR)/grad

$(OUT_DIR)/grad: $(GO_FILES)
	@mkdir -p $(OUT_DIR)
	go build -o $(OUT_DIR)/grad ./cmd/grad

# Build gractl binary
build-gractl: $(OUT_DIR)/gractl

$(OUT_DIR)/gractl: $(GO_FILES)
	@mkdir -p $(OUT_DIR)
	go build -o $(OUT_DIR)/gractl ./cmd/gractl

# Build for multiple platforms
build-all: build-linux build-darwin build-windows

build-linux:
	@mkdir -p $(OUT_DIR)
	GOOS=linux GOARCH=amd64 go build -o $(OUT_DIR)/grad-linux-amd64 ./cmd/grad
	GOOS=linux GOARCH=amd64 go build -o $(OUT_DIR)/gractl-linux-amd64 ./cmd/gractl

build-darwin:
	@mkdir -p $(OUT_DIR)
	GOOS=darwin GOARCH=amd64 go build -o $(OUT_DIR)/grad-darwin-amd64 ./cmd/grad
	GOOS=darwin GOARCH=amd64 go build -o $(OUT_DIR)/gractl-darwin-amd64 ./cmd/gractl
	GOOS=darwin GOARCH=arm64 go build -o $(OUT_DIR)/grad-darwin-arm64 ./cmd/grad
	GOOS=darwin GOARCH=arm64 go build -o $(OUT_DIR)/gractl-darwin-arm64 ./cmd/gractl

build-windows:
	@mkdir -p $(OUT_DIR)
	GOOS=windows GOARCH=amd64 go build -o $(OUT_DIR)/grad-windows-amd64.exe ./cmd/grad
	GOOS=windows GOARCH=amd64 go build -o $(OUT_DIR)/gractl-windows-amd64.exe ./cmd/gractl

# Clean build artifacts
clean:
	rm -rf $(OUT_DIR)
	go clean

# Run tests
test:
	go test ./...

# Generate protobuf code
generate:
	buf generate

# Minikube management
minikube-start:
	@echo "Starting minikube with 4C16G configuration..."
	minikube start --cpus=4 --memory=16384
	@echo "Setting Docker environment to use minikube's Docker daemon..."
	@eval $$(minikube docker-env)
	@echo "Verifying cluster status..."
	kubectl cluster-info

minikube-stop:
	@echo "Stopping minikube..."
	minikube stop

minikube-status:
	@echo "Minikube status:"
	minikube status
	@echo "Cluster info:"
	kubectl cluster-info

# Development workflow with Skaffold
dev: minikube-start
	@echo "Starting grad development mode with Skaffold..."
	@echo "Setting Docker environment..."
	@eval $$(minikube docker-env) && skaffold dev -p development --port-forward

dev-stop:
	@echo "Stopping Skaffold development mode..."
	@pkill -f "skaffold dev" || true

dev-debug: minikube-start
	@echo "Starting grad development mode with debug output..."
	@eval $$(minikube docker-env) && skaffold dev -p debug --port-forward -v debug

# Show help
help:
	@echo "Available targets:"
	@echo ""
	@echo "Build targets:"
	@echo "  build       - Build both binaries"
	@echo "  build-grad  - Build grad binary"
	@echo "  build-gractl- Build gractl binary"
	@echo "  build-all   - Build for all platforms"
	@echo "  clean       - Clean build artifacts"
	@echo "  test        - Run tests"
	@echo "  generate    - Generate protobuf code using buf"
	@echo ""
	@echo "Development targets:"
	@echo "  minikube-start   - Start minikube with 4C16G config"
	@echo "  minikube-stop    - Stop minikube"
	@echo "  minikube-status  - Show minikube and cluster status"
	@echo "  dev             - Start grad development mode (includes minikube start)"
	@echo "  dev-stop        - Stop development mode"
	@echo "  dev-debug       - Start development mode with debug output"
	@echo ""
	@echo "  help        - Show this help message"
