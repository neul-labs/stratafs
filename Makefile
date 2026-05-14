# Makefile for StrataFS

# Default target
.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: build
build: ## Build the application with FTS5 support
	mkdir -p build
	go build -o build/stratafs -tags "fts5" cmd/stratafs/main.go

.PHONY: run
run: ## Run the application with FTS5 support
	go run -tags "fts5" cmd/stratafs/main.go

.PHONY: install
install: ## Install the application
	go install -tags "fts5" cmd/stratafs/main.go

.PHONY: clean
clean: ## Clean build artifacts
	rm -rf build/

##@ Dependencies

.PHONY: deps
deps: ## Install dependencies
	go mod tidy

.PHONY: update
update: ## Update dependencies
	go get -u ./...
	go mod tidy

##@ Tools

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: fmt
fmt: ## Format code
	go fmt ./...

.PHONY: test
test: ## Run tests with FTS5 support
	go test -tags "fts5" -v ./...

.PHONY: fetch-onnx
fetch-onnx: ## Download the ONNX Runtime for the host platform (for local builds)
	bash scripts/get-onnx-runtime.sh

.PHONY: test-onnx
test-onnx: ## Run the full test suite with ONNX Runtime enabled
	bash scripts/test-with-onnx.sh

.PHONY: release
release: ## Build cross-platform release artifacts with bundled ONNX Runtime
	bash scripts/build-release.sh
