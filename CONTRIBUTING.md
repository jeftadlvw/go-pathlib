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
  file that compiles anywhere Go 1.26 does.
- The API stabilizes toward a v1 release through use in real projects.
  Breaking changes are allowed until then.

## General Implementation Instructions

- Read `CONVENTIONS.md` for code style, naming, testing, and commit
  conventions.
- Install [just](https://just.systems) and
  [golangci-lint](https://golangci-lint.run). `just` lists the recipes of the
  `Justfile`.
- Before opening a pull request, run `just fmt`, `just lint`,
  `just check-chars`, and `just test`.
- A change to the groups or to the code a group uses is verified with
  `just bundle`, which writes the bundles and compiles them.
- A change to a doc comment is followed by `just docs`, which updates
  `docs/pathlib.md`.
- CI runs the tests on Linux, macOS, and Windows with Go 1.26 and the latest
  stable Go. A change passes on all of them.
- Code converts a `Path` to a string only at the call that requires one. It
  passes `String()` to the operating system and `ToPosix()` to storage.

## Error Model

The library applies the error model for libraries of go-errkit
(`srcpad.dev/jeftadlvw/go-errkit`), defined in its `docs/LIBRARIES.md`. Read
that document first. This section names the parts in pathlib and where they
differ from the model. pathlib implements the model itself and imports no
package of go-errkit.

### Parts

| Part | In pathlib |
| --- | --- |
| Error kind type | `ErrorKind`, with the root error kind `ErrPathlib` |
| Error kinds | the `Err*` values in `core_error_kinds.go`, `fs_error_kinds.go`, and `io_error_kinds.go` |
| `wrapError` | gives a cause an error kind and returns the unexported `wrappedError` |
| Causes | the cause types `PathError` for most error kinds, `PatternError` below `ErrInvalidPattern`, `FileModeError` for `ErrInvalidFileMode`, and `OpenModeError` for `ErrInvalidOpenMode`, the `os` error for `ErrLookup`, and the error of the walk callback or the context for `ErrWalkInterrupted` |
| Descriptions | `slog.LogValuer` and `json.Marshaler` of `wrappedError` and of each cause type |

The cause types follow "Give a Cause a Type of Its Own" of the model. Most
errors carry the same data, the paths and an underlying error, so `PathError`
is their cause, independent of the group.

Code creates errors through the helpers built on `wrapError`:

- `pathErr(kind, paths...)` for an error that concerns paths.
- `osErr(operation, err, paths...)` for a failed call to the operating
  system. It picks the most specific error kind, see below.
- `osCreateErr(operation, err, path)` for a failed call to the operating
  system that creates `path`. A missing path then is its missing parent
  directory, so it returns `ErrParentNotExist`.
- `wrapErr(kind, cause, paths...)` for an error that concerns paths and is
  caused by another error, such as a check that finds a missing path with
  `wrapErr(ErrNotExist, fs.ErrNotExist, p)`.
- `patternErr(kind, pattern, cause, paths...)` for a rejected pattern.
- `fileModeErr(fileMode, paths...)` for a rejected file mode, and
  `openModeErr(openMode, paths...)` for a rejected open mode.
- `wrapError(ErrLookup, err)` for a failed lookup of a well-known directory.
- `wrapError(ErrWalkInterrupted, err)` for a walk that an error of its
  callback or a done context interrupts. Only `Walk` and `WalkR` wrap an
  error of the callback. The internal walk returns it unchanged, so a glob
  returns its own errors with their error kinds.

An error kind is never returned as an error on its own. Every error is
created through `wrapError` or one of the helpers above. An unexported
function that fails in one way only returns a `bool`, and its caller creates
the error.

### Text, Logs, and JSON

- **Codes.** Names are lowercase snake case, and a code joins them with dots
  from the root error kind, such as `pathlib.exist.file`. `defineError`
  panics for a name of another form.
- **Error text.** An error prints the message of its error kind, the code in
  parentheses, and the text of its cause, such as
  `path does not exist (pathlib.not_exist): a/b: open: no such file or directory`.
  Logs and JSON carry the code in a field of its own.
- **Code in parentheses.** The model puts the code in parentheses after the
  message, and so does pathlib. Programs on Unix print codes the same way,
  such as curl with `curl: (6) Could not resolve host` and Rust programs with
  `No such file or directory (os error 2)`. Go joins the layers of a wrapped
  error with colons, so a code between colons, such as
  `pathlib.not_exist: path does not exist: a/b: ...`, would read as one more
  layer and add a colon to a line that already has several. The parentheses
  keep the code a part of the message it belongs to.
- **Each path once.** A `PathError` names its paths, and the error of the
  `os` package below it names them again. `PathError.Error` leaves out a
  path that the underlying `*fs.PathError` or `*os.LinkError` repeats and
  keeps its operation and its reason, trimmed and with runs of whitespace
  collapsed. `Unwrap` returns the error unchanged, so `errors.Is` and
  `errors.AsType` see the original.
- **Paths in logs and JSON.** Error text shows paths in their native form.
  Logs and JSON show them in their Posix form, which is the form a `Path`
  serializes to.

### Differences from the Model

- **The most specific error kind.** Every error gets the most specific error
  kind the library knows. `osErr` turns an error of the operating system that
  matches `fs.ErrNotExist`, `fs.ErrExist`, or `fs.ErrPermission` into
  `ErrNotExist`, `ErrExist`, or `ErrPermissionDenied`. Any other error keeps
  the error kind of the operation, such as `ErrRead`. A check of the library
  that finds a missing or an existing path returns the same error kinds with
  `fs.ErrNotExist` or `fs.ErrExist` as the underlying error, such as
  `wrapErr(ErrNotExist, fs.ErrNotExist, p)`. A creation that finds the path
  created since its check runs the check again, so it returns the same error
  kind, such as `ErrFileExist`. Callers then check pathlib error kinds alone,
  and the `fs` sentinels still match through the chain.
- **Three families.** An error kind below `ErrInvalid` rejects a value the
  caller passed. An error kind below `ErrOperation` names an operation of the
  operating system that failed for a reason no condition describes. Every
  other error kind names a condition the library or the operating system
  found, such as `ErrNotExist`, and conditions of one sort share a group, such
  as `ErrWrongType`.
- **Error kinds per group.** The model keeps the error kind tree in one file.
  pathlib declares the error kinds of each group in its own
  `<group>_error_kinds.go`, so the `core` bundle stays self-contained. The
  family groups `ErrInvalid` and `ErrOperation` live in the `core` group, so
  every group can declare error kinds below them. The package comment in
  `core_path.go` and the README show the whole tree.
- **errors.AsType.** Callers read error kinds and causes with
  `errors.AsType`, as the model describes.

### Adding an Error Kind

1. Declare it with `defineError` in the error kinds file of the group that
   returns it, in its family and below the group callers handle it with. Its
   name is lowercase snake case.
2. Document its cause when it differs from `PathError`.
3. Add it to the tree in the package comment in `core_path.go`, to the tree
   in the README, and to `allKinds` in `core_error_test.go`.
4. Name it in the doc comment of every function that returns it.

Error kinds are part of the API. From v1 on, an error kind is never moved or
renamed, because its code and its group would change.
