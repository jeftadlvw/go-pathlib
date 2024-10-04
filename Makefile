.PHONY: test

test:
	@go test

doc-gen:
	@go install github.com/princjef/gomarkdoc/cmd/gomarkdoc@latest
	@gomarkdoc -o docs/pathlib.md -e .
