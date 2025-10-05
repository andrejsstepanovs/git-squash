# Binary name
BINARY=git-squash

# Setup go modules
export GO111MODULE=on

# Build the application
build: clean
	goreleaser build --single-target

# Install the application
install:
	go install main.go

# Run the application
run:
	go run main.go

# Clean built files
clean:
	rm -rf dist
	go clean
	rm -rf build

# Run tests
test:
	go test -v ./...

release: clean
	goreleaser release --rm-dist

# Build for different platforms
build-all: clean
	mkdir -p build
	GOOS=linux GOARCH=amd64 go build -o build/${BINARY}-linux-amd64 main.go
	GOOS=darwin GOARCH=amd64 go build -o build/${BINARY}-darwin-amd64 main.go
	GOOS=windows GOARCH=amd64 go build -o build/${BINARY}-windows-amd64.exe main.go

.PHONY: build install run clean test build-all
