# Binary name
BINARY=git-squash

# Setup go modules
export GO111MODULE=on

# Build the application
build:
	mkdir -p build
	go build -o build/${BINARY} main.go

# Install the application
install:
	go install main.go

# Run the application
run:
	go run main.go

# Clean built files
clean:
	go clean
	rm -rf build

# Run tests
test:
	go test -v ./...

# Build for different platforms
build-all: clean
	mkdir -p build
	GOOS=linux GOARCH=amd64 go build -o build/${BINARY}-linux-amd64 main.go
	GOOS=darwin GOARCH=amd64 go build -o build/${BINARY}-darwin-amd64 main.go
	GOOS=windows GOARCH=amd64 go build -o build/${BINARY}-windows-amd64.exe main.go

.PHONY: build install run clean test build-all
