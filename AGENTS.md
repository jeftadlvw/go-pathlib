# Agent Instructions

Include `README.md` for general information on the project and how users are expected to use it.
Include `CONTRIBUTING.md` for general contributing information.
Include `CONVENTIONS.md` for global implementation conventions.

This file provides additional rules and instructions for coding agents.
Some rules of included files apply differently to agents:
- Don't run tests, linting or formatting yourself. Developers will do these and prompt you to fix occurring issues.
- Write code so that section 3 (Formatting and Tooling) of `CONVENTIONS.md` passes without running the tools. When writing code, follow the conventions defined in `.golangci.yml`. If unknown, look them up first.

## General Constraints

- Agents tend to break the prose rules of `CONVENTIONS.md` section 1. Check every comment, Markdown file, and commit message against them before finishing. Em dashes are never allowed, and neither are semicolons or colons used to avoid them.
- Enforce LLM attribution in commit messages to be open about LLM usage. Add a trailer such as `Assisted-by: Claude Code (claude-opus-5-5)`.
- Never read, list, or edit files excluded by `.gitignore`.
- Never read binary files or files tracked by Git LFS (indicated by `.gitattributes`).
- Never edit generated files (`// Code generated ... DO NOT EDIT.`). Change the generator input and regenerate.
- Never add, remove, or upgrade dependencies unless asked.
- Report violations of the conventions found in unrelated code. Do not fix them inside the current change.

## Packaging and Distribution

- go-pathlib is distributed as a Go library module through Git tags (`v1.2.3`). Never create, move, or push tags.
- go-pathlib is also distributed as single-file bundles. `tools/bundle` merges the groups defined in `bundle.json` into `dist/`, and the release workflow attaches them to each release. Section 5 of `CONVENTIONS.md` defines the rules that keep every bundle compiling.
