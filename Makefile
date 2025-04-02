# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOTIDY=$(GOCMD) mod tidy
BINARY_NAME=lil
PKG=./...

.PHONY: all build run clean test tidy deps

all: build

# Build the application
build: deps
	$(GOBUILD) -o $(BINARY_NAME) .

# Run the application
run: build
	./$(BINARY_NAME)

# Run tests
test:
	$(GOTEST) -v $(PKG)

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

# Tidy dependencies
tidy:
	$(GOTIDY)

# Install dependencies (using go mod download and tidy)
deps:
	$(GOCMD) mod download
	$(GOTIDY) 