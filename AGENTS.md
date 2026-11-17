# Agent Instructions

Include `README.md` for general information on the project and how users are
expected to use it.
Include `CONTRIBUTING.md` for general contributing information.
Include `CONVENTIONS.md` for global implementation conventions.

This file provides additional rules and instructions for coding agents.
Some rules of included files apply differently to agents:
- Don't run tests, linting or formatting yourself. Developers will do these and
  prompt you to fix occurring issues.
- Write code so that section 3 (Formatting and Tooling) of `CONVENTIONS.md`
  passes without running the tools. When writing code, follow the conventions
  defined in `.golangci.yml`. If unknown, look them up first.

## General Constraints

- Agents tend to break the prose rules of `CONVENTIONS.md` section 1. Check
  every comment, Markdown file, and commit message against them before finishing.
  Em dashes are never allowed, and neither are semicolons or colons used to avoid
  them.
- Enforce LLM attribution in commit messages to be open about LLM usage. Add a
  trailer such as `Assisted-by: Claude Code (claude-opus-5-5)`.
- Never read, list, or edit files excluded by `.gitignore`.
- Never read binary files or files tracked by Git LFS (indicated by
  `.gitattributes`).
- Never edit generated files (`// Code generated ... DO NOT EDIT.`). Change the
  generator input and regenerate.
- Never add, remove, or upgrade dependencies unless asked.
- Report violations of the conventions found in unrelated code. Do not fix them
  inside the current change.

## Packaging and Distribution

- go-pathlib is distributed as a Go library module through Git tags (`v1.2.3`).
  Never create, move, or push tags.
- go-pathlib is also distributed as single-file bundles. `tools/bundle` merges
  the groups defined in `bundle.json` into `dist/`, and the release workflow
  attaches them to each release. Section 5 of `CONVENTIONS.md` defines the rules
  that keep every bundle compiling.
- `go.mod` names the oldest Go version the Go team supports. The library and
  every bundle target that version alone. There are no bundles per Go version,
  and source files carry no build constraints, which the bundler rejects.

## Planned Work

These items are kept for future versions. They are not part of a change unless
asked.

- **Per-platform code.** go-pathlib stays a library of one file per bundle, so
  it has no code that differs by operating system for now. Bundles allow no
  build constraints, and Go has no build constraints per function. Features
  that wait for a way to keep both:
  - The size, free space, and type of a filesystem (`statfs` on Unix, other calls
  on Windows).
  - File locks (`flock` on Unix, `LockFileEx` on Windows).
  - Opening a file without following symlinks (`O_NOFOLLOW`), which belongs to
  the support of `os.Root`.
- **Atomic checks and modifications.** A check of the state and the
  modification it guards should describe the same entry. The library should act
  on an open handle or use a call of the operating system that fails on its own,
  such as `O_EXCL`, in place of checking a path and then acting on it again. The
  known gap is the follow-up `Chmod` of a created directory. `openNonDir` already
  checks the opened file, and `Move` gives a file or a symlink its new name with
  a call that refuses an existing destination. Applying this to the whole library
  is planned for a future version, and the README roadmap lists it.
- **Umask and exact modes.** On Unix, the umask removes bits from `CreateMode`
  of `FileOptions`, `DirOptions`, and `OpenOptions`, and from `DefaultFileMode`
  and `DefaultDirMode`. The doc comments say so, and `SetMode` sets an exact
  mode. An `ExactMode` option could set the mode on the open file with
  `(*os.File).Chmod`. With `CreateIfNotExists` and no exclusive creation, an open
  cannot tell whether it created the file, so it cannot decide whether to apply
  the mode. `TempPathOptions` has no `CreateMode`, and `os.CreateTemp` creates
  with 0o600.
- **Atomic replacement of a file.** `ReplaceBytes`, `ReplaceString`, and their
  `WithOptions` variants would replace a file so that readers see the old or the
  new content and a crash keeps one of them. A temporary file with a name that
  starts with a dot is created next to the target, written, given its mode on the
  open file, synced, renamed over the target, and the directory is synced on
  Unix. Every failure removes the temporary file. It needs `os.CreateTemp` inside
  the library, because `CreateTempFileWithOptions` closes its file. A rename
  replaces a symlink at the target itself, breaks hard links, and drops extended
  attributes and ACLs. `writeBytes` and `copyFile` close the file without `Sync`,
  so their content is not on the disk yet when they return.
- **Owner of a path.** `SetOwner`, `SetOwnerWithOptions` with
  `OwnerOptions{NoFollow bool}`, and `FileInfo.Owner` would set and read the user
  and group IDs. `os.Chown` and `os.Lchown` compile on every platform, and
  `FileInfo.Owner` reads the fields `Uid` and `Gid` of `Sys()` by reflection, so
  no per-platform code is needed. The error kind for an unsupported platform
  cannot be named `unsupported` below `ErrWrongType`, because
  `ErrUnsupportedType` has that code. Options of the creating functions could later take an owner, so a file
  never exists with the wrong owner.
- **Support of `os.Root`.** Operations relative to an open `os.Root` keep every
  path inside the root, also through symlinks and "..". Wrapping them changes
  the API below `Path` at its base, so it waits for a release of its own.
  `Path.FS` wraps `os.DirFS`, which is no such boundary.
- **Breaking changes.** go-pathlib is before v1. Breaking changes are
  acceptable and need no deprecation period.
