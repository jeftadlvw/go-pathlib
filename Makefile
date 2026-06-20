.PHONY: test doc-gen bundle

test:
	@go test ./...

bundle:
	@go run ./tools/bundle

doc-gen:
	@go install github.com/princjef/gomarkdoc/cmd/gomarkdoc@latest
	@gomarkdoc -o docs/pathlib.md -e ./pathlib
