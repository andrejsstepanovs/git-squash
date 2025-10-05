# Binary name
BINARY=git-squash

# Setup go modules
export GO111MODULE=on

# Build the application
build: clean
	goreleaser release --snapshot --clean

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

# Run tests
test:
	go test -v ./...

release: clean
	git tag -a v$(shell grep 'const Version' version.go | awk '{print $$4}' | tr -d '"') -m "Release v$(shell grep 'const Version' version.go | awk '{print $$4}' | tr -d '"')"
	git push --tags
	goreleaser release

.PHONY: build install run clean test
