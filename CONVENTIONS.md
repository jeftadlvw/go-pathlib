# Conventions

These conventions define how code, comments, and commits are written. They
apply equally to humans and agents. Where they are silent, follow these in
order:
[Go Code Review Comments](https://go.dev/wiki/CodeReviewComments),
[Google Go Style Guide](https://google.github.io/styleguide/go/),
[Effective Go](https://go.dev/doc/effective_go).

These conventions take precedence over patterns found in existing code.

## 1. Prose

Applies to comments, doc comments, Markdown, panic and error messages, and
commit messages.

- American English.
- ASCII punctuation only. No em dashes, en dashes, curly quotes, or the
  ellipsis character.
- `-` is a hyphen in compound words (`read-only`) or a Markdown list marker.
  It is never a sentence dash, and neither is `--`.
- No semicolons joining clauses. Write two sentences.
- No colons joining clauses. A colon only introduces a list, an example, a
  code sample, or follows a label.
- State what something is and does. Do not define it by contrast ("unlike X",
  "rather than Y", "not a Z").
- No change narration. Text describes the code as it is, not its history
  ("now", "new", "previously", "changed to", "fixed").
- One idea per sentence. Cut words that do not change the meaning.

## 2. Comments

- Every exported identifier has a doc comment. Struct fields, unexported
  types, and non-trivial unexported functions have one too.
- Doc comments are full sentences. They start with the identifier's name and
  end with a period.
- Doc comments follow this structure, stopping when nothing remains to say:
    1. Summary sentence. What it is or does.
    2. Elaboration. Behavior, invariants, ownership of arguments and results.
    3. Special cases. Nil, empty and zero inputs, returned errors, panics,
       concurrency safety.
    4. Example, only when usage is not obvious. Prefer an `Example` test.
- Doc comments do not describe callers, other packages, or where the
  identifier is used.
- Doc comments refer to the receiver in words ("the client", "the client's
  timeout"), never by the receiver's name. Parameters are referred to by
  name, so exported functions give them descriptive names. An abbreviated
  parameter such as `err` is spelled out ("the error").
- Doc comments link the first mention of every other exported identifier
  with a [doc link](https://go.dev/doc/comment#doclinks), as in `[NewClient]`,
  `[Client.Do]`, or `[errors.Is]`. The documented identifier, later mentions,
  and code expressions such as `errors.As(err, &kind)` stay plain. Field and
  inline comments have no links.
- Inline comments explain why, never what. Delete comments that restate the
  code.
- `TODO` comments reference an issue, as in `// TODO(#123): ...`. No bare
  `TODO`, `FIXME`, or `XXX`.
- No commented-out code.
- Wrap comments at 80 columns.
- Every Go file starts with a short comment that names the file and says
  what it holds. A blank line separates it from the package clause, so it
  never becomes the package comment. The file that carries the package
  comment, test files, and files under `testdata/` need none.

  ```go
  // core_compare.go holds the lexical comparison of paths.

  package pathlib
  ```

Section comments group declarations at file level and use the blank-line
style. No filler characters such as `// ----- X -----` or `// ===== X =====`.

```go
//
// Construction
//
```

A function that needs section comments is split instead.

## 3. Formatting and Tooling

- `gofmt` and `goimports` clean. Imports are grouped as standard library,
  third party, then module-local.
- `golangci-lint run` passes with the configuration in `.golangci.yml`.
- `go test ./...` passes.
- The Go version in `go.mod`, Go 1.22, is the language baseline. Use the
  standard library of that version (`slices`, `maps`, `min`, `max`,
  `for range n`) over hand-written equivalents.
- CI rejects the forbidden characters from section 1 in Go and Markdown
  files. `make check-chars` runs the same check.

The Makefile wraps every tool. `make install-dev-tools` installs them.

## 4. Naming

- MixedCaps. Initialisms keep a single case: `ID`, `URL`, `HTTP`, `FS`,
  `JSON` (`userID`, `ServeHTTP`, `DirFS`).
- Package names are short, lowercase, singular, and one word. No `util`,
  `common`, `misc`, `helpers`, `types`, or `models`.
- No stutter. `config.Load`, not `config.LoadConfig`. `list.List` is fine
  when the type is the package's subject.
- Getters are named after the value (`Name()`), never `GetName()`.
- Constructors are `New<Type>`, or `New` for the package's primary type.
  Constructors that panic on invalid input are prefixed with `Must`.
- Receivers are one or two letters and identical across all methods of a
  type. Never `this` or `self`.
- Booleans read as predicates: `isDir`, `hasCause`, `Enabled`.
- Abbreviations only where established: `ctx`, `cfg`, `err`, `buf`, `i`,
  `n`.

## 5. Files and Packages

- Source files in `pathlib/` are named `<group>.go` or
  `<group>_<concern>.go`, such as `io.go` or `core_compare.go`. The groups
  and their prefixes are defined in `bundle.json`. `tools/bundle` merges the
  files of the groups into single-file bundles.
- Code of the `core` group uses no code of another group, so the `core`
  bundle compiles on its own. The other groups are bundled together and may
  use each other.
- A file holds one concern, such as the comparison of paths. A type lives in
  the file of its concern, together with its constructors and methods. A
  type that serves several concerns gets a file named after it, such as
  `Path` in `core_path.go`.
- One enum per file, holding its type, constants, and methods.
- Unexported helpers shared by several files live in a file named after
  their concern (`core_validate.go`). Never `util.go` or `helpers.go`.
- `core_path.go` carries the package comment, so every bundle includes it.
  Exactly one file carries the package comment.
- Order within a file: constants, variables, types, constructors, exported
  functions, exported methods, unexported methods, unexported functions.
- The library is the single package `pathlib`, because each bundle is a
  single file. Code not meant for importers is unexported.
- Tests sit beside the code. `core_path.go` is tested in
  `core_path_test.go`.

## 6. Types and Data Ownership

### Enums

- Typed constants on a named type. The underlying type is `string` when the
  value is serialized or shown to users, and `int` with `iota` otherwise.
- The zero value is invalid, or a named constant with a documented meaning.
  That is an absent value such as `Unknown` or `None`, or the default of an
  option.
- Every enum has `Valid() bool`. Integer enums also have `String()`.
- Switches over an enum are exhaustive (`exhaustive` linter).

### Value Types

Types that are built once and only read afterwards.

- Fields are unexported. Reading happens through getters.
- The constructor validates its inputs and rejects invalid ones. A
  constructed value is always valid.
- The constructor clones every slice and map it receives. Getters return
  clones of slices and maps, never the internal ones.
- No setters. A modified variant is a separate value, returned by a
  `With<Field>` method.

### Configuration Structs

- Fields are exported, so callers build them with a literal.
- Every field's zero value is valid with a documented meaning, or the
  consumer rejects it.
- Consumers take configuration by value (`func New(cfg Config)`), never by
  pointer.
- Copying a struct shares its slices, maps, and pointers. The consumer
  clones every such field it retains.

### General

- A type uses either pointer or value receivers for all of its methods.
- Accept interfaces and return concrete types. Interfaces are declared in the
  consuming package and stay small.
- Functions that fail return the `error` interface, never a concrete error
  type. A nil `*ParseError` stored in an `error` does not compare equal to
  `nil`. Constructors of error values return their concrete type.
- Exported structs embed a type only when its promoted methods are intended
  API.
- Generics only where they remove real duplication.

## 7. Errors

- Errors are declared, created, wrapped, and inspected as the Error Model
  section of `CONTRIBUTING.md` describes. It applies the error model for
  libraries of go-errkit (`srcpad.dev/jeftadlvw/go-errkit`), defined in its
  `docs/LIBRARIES.md`. The model uses the standard library alone.
- `panic` is reserved for programmer errors. Runtime input never causes a
  panic.
- Panic messages start with the qualified name of the failing function
  (`client.New: ...`) and name the violated rule and the offending value.
- Never match on `Error()` strings.
- An error is handled once. It is logged or returned, never both.

## 8. Context, Concurrency, and State

- `context.Context` is the first parameter, named `ctx`. It is never stored
  in a struct and never `nil`.
- Every goroutine has an owner that waits for it to finish (`sync.WaitGroup`,
  `errgroup`).
- Mutable package-level state is forbidden. Package level holds only
  constants, sentinel errors, and values built during initialization and
  never modified afterwards.
- No `init()` functions. Package variables are initialized in their
  declaration.

## 9. Control Flow

- Return early. No `else` after a `return`.
- `if` and `switch` statements have no init statement. Assign the value on
  its own line, then test it. `noinlineerr` enforces this for errors only.

  ```go
  entry, ok := cache[key]
  if ok {
  ```
- No naked returns. Named results only when they document a result or a
  deferred function needs them.
- A function does one thing at one level of abstraction.

## 10. Testing

- Use `github.com/stretchr/testify/require`, never `assert`. A failed check
  stops the test.
- Check errors with `require.ErrorIs` against kinds, such as `ErrNotExist`.
  Never compare error strings. Tests of rendering are the exception.
- Tests are table-driven with named cases run through `t.Run`.
- Tests and subtests call `t.Parallel()`.
- Tests of public behavior use the external test package
  (`package client_test`).
- Use `t.TempDir()` for files and `context.Background()` for contexts.
  Fixtures live in `testdata/`. Never synchronize with `time.Sleep`.
- Test names are `Test<Type>_<Method>` or `Test<Function>`.
- Examples live in `example_test.go` and carry an `// Output:` comment.

## 11. Dependencies

- The library has no runtime dependencies. Each bundle is a single file
  that compiles with the standard library alone.
- Test-only dependencies are limited to `testify`.
- `go mod tidy` leaves no diff.

## 12. Commits

[Conventional Commits](https://www.conventionalcommits.org):
`<type>(<scope>): <description>`

Types: `feat`, `fix`, `refactor`, `perf`, `test`, `docs`, `style`, `build`,
`ci`, `chore`, `revert`.

- The scope is optional and encouraged for localized changes. It names a
  type, file, or area.
- The description is imperative, at most 72 characters, and has no trailing
  period. It is lowercase except for identifiers and proper nouns.
- Breaking changes add `!` after the type or scope and a `BREAKING CHANGE:`
  footer.
- The body explains why, wrapped at 72 columns.
- One logical change per commit. Style fixes to unrelated code go into a
  separate `style` commit.
- Commits containing agent-written content carry an attribution trailer.

```
feat(logging): add log level filtering
fix(timer): guard startTime field against unstarted timer
chore: bump foobarbaz dependency

feat(client)!: reject negative retry limits

A negative limit disabled retries without a visible cause.

BREAKING CHANGE: NewClient returns an error for a negative retry limit.
```
