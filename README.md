# reuse

Go library for parsing REUSE-compliant projects (spec v3.3). Extracts SPDX license and copyright information from file headers, .license sidecars, REUSE.toml annotations, and .reuse/dep5 files.

SPDX expressions are stored as raw strings. Validation is left to consumers (see [github.com/git-pkgs/spdx](https://github.com/git-pkgs/spdx) if you need it).

## Installation

```bash
go get github.com/git-pkgs/reuse
```

## Usage

### Parse a project

```go
import "github.com/git-pkgs/reuse"

p, err := reuse.OpenProject("/path/to/repo")

// Get licensing info for a single file.
info, err := p.ReuseInfoOf("src/main.go")
fmt.Println(info.LicenseExpressions) // ["MIT"]
fmt.Println(info.CopyrightNotices)   // ["2024 Jane Doe <jane@example.com>"]

// Walk all covered files.
all, err := p.AllReuseInfo()
for path, info := range all {
    fmt.Printf("%s: %v\n", path, info.LicenseExpressions)
}
```

### Extract SPDX tags from text

```go
info := reuse.ExtractReuseInfo(`// SPDX-License-Identifier: MIT OR Apache-2.0
// SPDX-FileCopyrightText: 2024 Alice`)

fmt.Println(info.LicenseExpressions) // ["MIT OR Apache-2.0"]
fmt.Println(info.CopyrightNotices)   // ["2024 Alice"]
```

### Parse REUSE.toml

```go
rt, err := reuse.ParseReuseTOMLFile("REUSE.toml")

info, precedence, ok := rt.ReuseInfoOf("src/main.go")
if ok {
    fmt.Println(info.LicenseExpressions)
    fmt.Println(precedence) // "closest", "aggregate", or "override"
}
```

### Parse .reuse/dep5

```go
d, err := reuse.ParseDep5File(".reuse/dep5")

info, ok := d.ReuseInfoOf("docs/guide.md")
if ok {
    fmt.Println(info.LicenseExpressions)
}
```

## Development

```bash
git clone --recurse-submodules https://github.com/git-pkgs/reuse
go test -v -race ./...
```

## License

MIT
