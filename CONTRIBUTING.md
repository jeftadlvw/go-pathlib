# Contributing to go-pathlib

## About the Project

go-pathlib is a Go library for handling filesystem paths. Its API is
inspired by Python's [pathlib](https://docs.python.org/3/library/pathlib.html).
An immutable `Path` holds a path in a canonical Posix form and converts it to
the native form of the platform at the boundary to the operating system.

The library ships in two forms:

- A Go module, imported as `github.com/jeftadlvw/go-pathlib/pathlib`.
- Single-file bundles that users copy into their projects. The release
  workflow attaches them to every release.

| Path | Holds |
| --- | --- |
| `pathlib/` | the library, a single package split into groups by filename prefix |
| `bundle.json` | the groups and the bundles built from them |
| `tools/bundle/` | the bundler, which merges the groups into single files |
| `docs/pathlib.md` | the API reference, generated from the doc comments |
| `dist/` | the generated bundles, ignored by Git |

## Goal

- One immutable `Path` type is the source of truth for every filesystem path.
- A path behaves the same on Linux, macOS, and Windows.
- Output is platform-native by default and explicit when needed.
- Behavior is strict by default and flexible on request, such as case
  sensitivity.
- The library needs the standard library alone, so every bundle is a single
  file that compiles anywhere Go 1.22 does.
- The API stabilizes toward a v1 release through use in real projects.
  Breaking changes are allowed until then.

## General Implementation Instructions

- Read `CONVENTIONS.md` for code style, naming, testing, and commit
  conventions.
- Run `make install-dev-tools` once to install golangci-lint and gomarkdoc.
- Before opening a pull request, run `make fmt`, `make lint`,
  `make check-chars`, and `make test`.
- A change to the groups or to the code a group uses is verified with
  `make bundle` followed by `go build ./dist/...`.
- A change to a doc comment is followed by `make generate-docs`, which
  updates `docs/pathlib.md`.
- CI runs the tests on Linux, macOS, and Windows with Go 1.22 and the latest
  stable Go. A change passes on all of them.
- Code converts a `Path` to a string only at the call that requires one. It
  passes `String()` to the operating system and `ToPosix()` to storage.

## Error Model

The library applies the error model for libraries of go-errkit
(`srcpad.dev/jeftadlvw/go-errkit`), defined in its `docs/LIBRARIES.md`. Read
that document first. This section names the parts in pathlib and where they
differ from the model.

### Parts

| Part | In pathlib |
| --- | --- |
| Kind type | `PathlibError`, with the root `ErrPathlib` |
| Kinds | the `Err*` values in `core_error_kinds.go`, `fs_error_kinds.go`, and `io_error_kinds.go` |
| Causes | `PathError` for most kinds, `PatternError` below `ErrInvalidPattern`, `FileModeError` for `ErrInvalidFileMode`, `OpenModeError` for `ErrInvalidOpenMode`, and the `os` error for `ErrLookup` |
| Failure | the unexported `wrappedError`, created by `wrapError` |

Code creates failures through the helpers built on `wrapError`:

- `pathErr(kind, paths...)` for a failure that concerns paths.
- `osErr(operation, err, paths...)` for a failed call to the operating
  system. It picks the most specific kind, see below.
- `osCreateErr(operation, err, path)` for a failed call to the operating
  system that creates `path`. A missing path then is its missing parent
  directory, so it returns `ErrParentNotExist`.
- `wrapErr(kind, cause, paths...)` for a failure caused by another error that
  is not an error of the operating system, such as an error of a walk
  callback.
- `patternErr(kind, pattern, cause, paths...)` for a rejected pattern.
- `fileModeErr(fileMode, paths...)` for a rejected file mode, and
  `openModeErr(openMode, paths...)` for a rejected open mode.
- `raiseError(ErrLookup, err)` for a failed lookup of a well-known directory.

A kind is never returned as an error on its own. Every failure is created
through `wrapError` or one of the helpers above. An unexported function that
fails in one way only returns a `bool`, and its caller creates the failure.

### Differences from the Model

- **The most specific kind.** Every failure gets the most specific kind the
  library knows. `osErr` turns an error of the operating system that matches
  `fs.ErrNotExist`, `fs.ErrExist`, or `fs.ErrPermission` into `ErrNotExist`,
  `ErrExist`, or `ErrPermissionDenied`. Any other error keeps the kind of the
  operation, such as `ErrRead`. A check of the library that finds a missing or
  an existing path returns the same kinds with `fs.ErrNotExist` or
  `fs.ErrExist` as the underlying error, such as
  `wrapErr(ErrNotExist, fs.ErrNotExist, p)`. A creation that finds the path
  created since its check runs the check again, so it returns the same kind,
  such as `ErrFileExist`. Callers then check pathlib kinds alone, and the `fs`
  sentinels still match through the chain.
- **Three families.** A kind below `ErrInvalid` rejects a value the caller
  passed. A kind below `ErrOperation` names an operation of the operating
  system that failed for a reason no condition describes. Every other kind
  names a condition the library or the operating system found, such as
  `ErrNotExist`, and conditions of one sort share a group, such as
  `ErrWrongType`.
- **Kinds per group.** Each group declares the kinds it returns in its own
  `<group>_error_kinds.go`, so the `core` bundle stays self-contained. The
  family groups `ErrInvalid` and `ErrOperation` live in the `core` group, so
  every group can declare kinds below them. The README shows the whole tree.
- **Code format.** Names are UPPER_SNAKE_CASE, and a code joins them with
  dots, such as `PATHLIB.EXIST.FILE`.
- **Error text.** A failure prints the message of its kind, the code in
  parentheses, and the text of its cause, such as
  `path does not exist (PATHLIB.NOT_EXIST): a/b: open a/b: no such file or directory`.
  Parentheses read naturally in plain output. Logs and JSON carry the code
  in a field of its own.
- **One shared cause.** Most failures carry the same data, the paths and an
  underlying error. `PathError` is their cause, independent of the group.
- **errors.As.** Callers read kinds and causes with `errors.As`.
  `errors.AsType` needs Go 1.26, above the baseline of the library.
- **Paths in logs and JSON.** Error text shows paths in their native form.
  Logs and JSON show them in their Posix form, which is the form a `Path`
  serializes to.

### Adding a Kind

1. Declare it with `defineError` in the kinds file of the group that returns
   it, in its family and below the group callers handle it with.
2. Document its cause when it differs from `PathError`.
3. Add it to the tree in the README and to `allKinds` in
   `core_error_test.go`.
4. Name it in the doc comment of every function that returns it.

Kinds are part of the API. From v1 on, a kind is never moved or renamed,
because its code and its group would change.
