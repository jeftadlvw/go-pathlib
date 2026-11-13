# Justfile drives building, testing, linting, bundling and documenting the
# library.

# Version of gomarkdoc, which `just docs` runs.
gomarkdoc_version := "v1.1.0"

# The repository the API reference links to. Passing it explicitly keeps the
# generated links independent of the remote of the local clone.
repository_url := "https://github.com/jeftadlvw/go-pathlib"

# Lists the available recipes.
default:
    @just --list

# Compiles every package, for example `just build ./pathlib/...`.
build *packages="./...":
    go build {{packages}}

# Runs Go tests with the race detector, for example `just test ./pathlib/...`.
test *packages="./...":
    go test -race {{packages}}

# Runs golangci-lint, for example `just lint ./tools/...`.
lint *packages="./...":
    golangci-lint run {{packages}}

# Runs golangci-lint and applies its automatic fixes.
lint-fix *packages="./...":
    golangci-lint run --fix {{packages}}

# Formats every Go file with the formatters in .golangci.yml.
fmt:
    golangci-lint fmt

# Rejects en dashes, em dashes, curly quotes, and the ellipsis character in Go and Markdown files, as CI does.
check-chars:
    @! LC_ALL=C git grep --untracked -nP '\xE2\x80[\x93\x94\x98\x99\x9C\x9D\xA6]' -- '*.go' '*.md'

# Writes the single-file bundles of bundle.json into dist/ and compiles them.
bundle:
    go run ./tools/bundle
    go build ./dist/...

# Generates the API reference into the embed tags of docs/pathlib.md.
docs:
    go run github.com/princjef/gomarkdoc/cmd/gomarkdoc@{{gomarkdoc_version}} \
        --repository.url {{repository_url}} \
        --repository.default-branch main \
        --repository.path / \
        --embed \
        --output docs/pathlib.md \
        ./pathlib
