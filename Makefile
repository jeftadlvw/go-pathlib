.PHONY: install-dev-tools format fmt lint lint-fix check-chars test bundle generate-docs

# Install the development tools
install-dev-tools:
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	go install github.com/princjef/gomarkdoc/cmd/gomarkdoc@latest

# Format the code
format:
	golangci-lint fmt ./...

fmt: format

# Run the linters
lint:
	golangci-lint run ./...

# Run the linters and apply automatic fixes
lint-fix:
	golangci-lint run --fix ./...

# Reject en dashes, em dashes, curly quotes, and the ellipsis character in Go
# and Markdown files
check-chars:
	@! LC_ALL=C git grep --untracked -nP '\xE2\x80[\x93\x94\x98\x99\x9C\x9D\xA6]' -- '*.go' '*.md'

# Run all tests
test:
	go test ./...

# Create single-file, isolated source code library bundles
bundle:
	go run ./tools/bundle

# Generate API documentation into the embed tags of docs/pathlib.md
generate-docs:
	gomarkdoc --embed --output docs/pathlib.md ./pathlib
