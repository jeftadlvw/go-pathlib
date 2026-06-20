.PHONY: test doc-gen

test:
	@go test

doc-gen:
	@go install github.com/princjef/gomarkdoc/cmd/gomarkdoc@latest
	@gomarkdoc -o docs/pathlib.md -e ./pathlib
