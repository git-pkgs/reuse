# reuse

Go library for reading REUSE licensing metadata from file headers, `.license`
sidecars, `REUSE.toml` annotations, and `.reuse/dep5` files. SPDX expressions
stay as raw strings; use [spdx](https://github.com/git-pkgs/spdx) to validate
them. The package does not check whether a project is REUSE compliant.

```sh
go get github.com/git-pkgs/reuse
```

## Read a project

```go
import (
    "fmt"
    "log"

    "github.com/git-pkgs/reuse"
)

project, err := reuse.OpenProject("/path/to/repo")
if err != nil {
    log.Fatal(err)
}
info, err := project.ReuseInfoOf("src/main.go")
if err != nil {
    log.Fatal(err)
}
fmt.Println(info.LicenseExpressions)
fmt.Println(info.CopyrightNotices)
```

`OpenProject` loads annotation files once. `Project.ReuseTOMLs` maps relative
directory names to parsed TOML files, with `"."` for the root. Each lookup reads
the source file or its sidecar; reopen the project after changing annotations.
Paths passed to `ReuseInfoOf` must be relative to the project root.

A sidecar replaces the source file's header, then participates in annotation
resolution. The outermost matching `override` suppresses closer annotations
and both header and sidecar. Otherwise, the nearest `closest` values fill
missing licence and copyright fields separately. `aggregate` adds values;
ancestor aggregates still apply above an override. Nested TOML patterns are
relative to their own directory, and the last matching table within one file
is used. These rules follow the [REUSE specification](https://reuse.software/spec-3.3/#reuse-toml).

When no TOML files are present, `.reuse/dep5` entries are aggregated with the
header or sidecar. `SourcePath` and `SourceType` identify the primary source;
the combined result does not retain provenance for every individual value.

`project.AllReuseInfo()` returns a map for covered files, including empty
results when no metadata was found. The walk skips symlinks, empty files,
licence files, sidecars, SPDX documents, and the built-in ignored directories.
It does not apply `.gitignore` or exclude VCS submodules and Meson subprojects.
Use individual lookups when the caller has its own file-selection policy.

## Read text or annotation files

Text extraction is in the `extract` subpackage:

```go
import "github.com/git-pkgs/reuse/extract"

info := extract.ExtractReuseInfo(`// SPDX-License-Identifier: MIT OR Apache-2.0
// SPDX-FileCopyrightText: 2024 Alice`)
fmt.Println(info.LicenseExpressions) // [MIT OR Apache-2.0]
```

`ExtractReuseInfo` requires complete text and treats its final line as
terminated. Passing a prefix ending at `SPDX-License-Identifier: MIT` can
therefore lose a later `OR Apache-2.0`. Use `peek` for bounded-prefix claims,
and keep those observations separate from resolved project metadata.

The extractor handles REUSE ignore blocks and collects snippet declarations
alongside file declarations. It does not retain snippet scopes or byte spans.
Empty tags are ignored, and values do not continue onto subsequent lines.
Returned values own their strings, so retaining a short header does not retain
the whole input. `extract.ExtractFromFile` reads the complete file and returns
empty metadata for NUL bytes or invalid UTF-8 in its first 8 KiB, extending that
probe just enough to finish a crossing code point. UTF-16 is not decoded.

The `toml` subpackage reads annotations without opening a project:

```go
import "github.com/git-pkgs/reuse/toml"

config, err := toml.ParseReuseTOMLFile("REUSE.toml")
if err != nil {
    log.Fatal(err)
}
info, precedence, ok := config.ReuseInfoOf("src/main.go")
if ok {
    fmt.Println(info.LicenseExpressions, precedence)
}
```

For DEP5, use `dep5.ParseDep5File` from `github.com/git-pkgs/reuse/dep5`.
Both subpackages also provide string parsers, `toml.ParseReuseTOML` and
`dep5.ParseDep5`. Parsed examples are included in the Go tests.

File reads and traversal have no size or file-count limits. `AllReuseInfo`
retains all results; applications processing large trees can use individual
lookups to consume results as they go. Concurrent lookups are supported while
the project and its parsed annotations remain unchanged.

## Development

Initialize the example fixture after cloning, then run the tests and benchmarks:

```sh
git submodule update --init
go test -race ./...
go test ./extract -run '^$' -bench BenchmarkExtractReuseInfo -benchmem
go test ./extract -run '^$' -bench BenchmarkRetainedHeaders -benchtime 1x
go test . -run '^$' -bench BenchmarkProject -benchmem
```

The extraction benchmarks cover short and long inputs, ignore blocks, and
snippets. `BenchmarkRetainedHeaders` reports heap retained after collecting
short licence values from separate 128 KiB inputs. `BenchmarkProject` includes
file reads and annotation resolution; its results depend on filesystem caches.

## License

[MIT](LICENSE).
