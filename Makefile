.PHONY: install-tools test doc-gen bundle

install-dev-tools:
	@go install github.com/princjef/gomarkdoc/cmd/gomarkdoc@latest

test:
	@go test ./...

bundle:
	@go run ./tools/bundle

generate-docs:
	@gomarkdoc -o docs/pathlib.md -e ./pathlib
