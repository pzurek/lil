# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOTIDY=$(GOCMD) mod tidy
BINARY_NAME=lil
PKG=./...
BIN_DIR=bin
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')

# Phony targets
.PHONY: all build run clean test tidy deps schema gen linear install fmt vet lint check help release-homebrew

# Default target
all: build

# Help target
help:
	@echo "Available targets:"
	@echo "  all       : Default target, builds the application"
	@echo "  build     : Build the application"
	@echo "  run       : Build and run the application"
	@echo "  clean     : Clean build artifacts"
	@echo "  tidy      : Tidy go modules"
	@echo "  deps      : Install dependencies" 
	@echo "  schema    : Fetch Linear GraphQL schema"
	@echo "  gen       : Generate Go code from GraphQL schema"
	@echo "  linear    : Run all Linear GraphQL generation steps"
	@echo "  install   : Install the application to /usr/local/bin"
	@echo "  test      : Run tests"
	@echo "  fmt       : Format code"
	@echo "  vet       : Run go vet"
	@echo "  lint      : Run golangci-lint"
	@echo "  check     : Run all code quality checks"
	@echo "  release-homebrew : Create a new release and update Homebrew formula"

# Build the application
build: deps
	@echo "Building $(BINARY_NAME) version $(VERSION)..."
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) -ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)" -o $(BIN_DIR)/$(BINARY_NAME) .
	@echo "✅ Build complete: $(BIN_DIR)/$(BINARY_NAME)"

# Run the application
run: build
	@echo "Running $(BINARY_NAME)..."
	$(BIN_DIR)/$(BINARY_NAME)

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	$(GOCLEAN)
	@rm -rf $(BIN_DIR)
	@echo "✅ Clean complete"

# Tidy dependencies
tidy:
	$(GOTIDY)

# Install dependencies (using go mod download and tidy)
deps:
	@echo "Installing dependencies..."
	$(GOCMD) mod download
	$(GOTIDY)
	@echo "✅ Dependencies installed"

# --- Linear GraphQL Schema & Client Generation ---

.PHONY: schema
schema:
	@if [ -z "$$LINEAR_API_KEY" ]; then \
		echo "❌ Error: LINEAR_API_KEY environment variable is not set"; \
		echo "Please set it with: export LINEAR_API_KEY=your_api_key"; \
		exit 1; \
	fi
	@if ! command -v geq &> /dev/null; then \
		echo "⚠️ geq command not found. Installing..."; \
		go install github.com/pzurek/geq@latest; \
	fi
	@echo "Fetching Linear GraphQL schema using geq..."
	@mkdir -p internal/linear/schema # Ensure directory exists
	@geq -e https://api.linear.app/graphql \
		-H "Authorization:$$LINEAR_API_KEY" \
		-o internal/linear/schema/schema.graphql \
		-m
	@if [ -f "schema.min.graphql" ]; then \
		mv schema.min.graphql internal/linear/schema/schema.min.graphql; \
		echo "✅ Schema fetched successfully"; \
	else \
		echo "❌ Error: Failed to generate schema.min.graphql"; \
		exit 1; \
	fi

.PHONY: gen
gen:
	@echo "Generating Go code from GraphQL schema..."
	@if [ ! -f internal/linear/schema/genqlient.yaml ]; then \
		echo "❌ Error: genqlient.yaml configuration file not found in internal/linear/schema directory"; \
		echo "Please create a configuration file for genqlient"; \
		exit 1; \
	fi
	@if [ ! -f internal/linear/schema/operations.graphql ]; then \
		echo "❌ Error: operations.graphql file not found in internal/linear/schema directory"; \
		echo "Please create the operations.graphql file with your GraphQL queries"; \
		exit 1; \
	fi
	@echo "Running genqlient..."
	@cd internal/linear/schema && go run github.com/Khan/genqlient
	@echo "✅ genqlient complete"

# Note: The original Makefile had a 'linear.go' at the top level, which might conflict
# if genqlient is also configured to output there. Assuming genqlient outputs
# 'generated.go' inside internal/linear/schema as seen in the listing.
.PHONY: linear
linear: schema gen
	@echo "✅ All GraphQL generation steps completed successfully"


# --- Installation ---

.PHONY: install
install: build
	@echo "Installing $(BINARY_NAME) to /usr/local/bin/$(BINARY_NAME)..."
	@if [ ! -d /usr/local/bin ]; then \
		echo "❌ Error: /usr/local/bin directory does not exist"; \
		exit 1; \
	fi
	@if [ -w /usr/local/bin ]; then \
		cp $(BIN_DIR)/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME); \
		echo "✅ Installation complete! You can now use '$(BINARY_NAME)' from any directory."; \
	else \
		echo "⚠️ You don't have write permission to /usr/local/bin"; \
		echo "Try running: sudo make install"; \
		exit 1; \
	fi


# --- Testing & Code Quality ---

.PHONY: test
test:
	@echo "Running tests..."
	@$(GOTEST) -v $(PKG) || { echo "❌ Tests failed"; exit 1; }
	@echo "✅ Tests passed"

.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	@mkdir -p $(BIN_DIR)
	@$(GOTEST) -coverprofile=$(BIN_DIR)/coverage.out $(PKG) || { echo "❌ Tests failed"; exit 1; }
	@$(GOCMD) tool cover -html=$(BIN_DIR)/coverage.out -o $(BIN_DIR)/coverage.html
	@echo "✅ Coverage report generated at $(BIN_DIR)/coverage.html"

.PHONY: fmt
fmt:
	@echo "Formatting code..."
	@go fmt $(PKG)
	@echo "✅ Formatting complete"

.PHONY: vet
vet:
	@echo "Vetting code..."
	@go vet $(PKG) || { echo "❌ Vetting failed"; exit 1; }
	@echo "✅ Vetting complete"

.PHONY: lint
lint:
	@if ! command -v golangci-lint &> /dev/null; then \
		echo "⚠️ golangci-lint not found, installing..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	@echo "Linting code..."
	@golangci-lint run || { echo "❌ Linting failed"; exit 1; }
	@echo "✅ Linting complete"

.PHONY: check
check: fmt vet lint test
	@echo "✅ All checks passed"

# --- Release for Homebrew ---

.PHONY: release-homebrew
release-homebrew:
	@if [ -z "$(TAG)" ]; then \
		echo "❌ Error: TAG environment variable is not set"; \
		echo "Please set it with: make release-homebrew TAG=v0.1.0"; \
		exit 1; \
	fi
	@echo "Creating release $(TAG) for Homebrew..."
	@./scripts/release.sh $(TAG)
	@echo "✅ Homebrew release process complete"
