package dep5

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/git-pkgs/reuse/internal/core"
)

func TestParseDep5_Valid(t *testing.T) {
	content := `Format: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/

Files: src/*
Copyright: 2024 Alice
License: MIT

Files: doc/*
Copyright: 2023 Bob
License: CC0-1.0
`

	d, err := ParseDep5(content)
	if err != nil {
		t.Fatal(err)
	}

	if d.Header.Format != "https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/" {
		t.Errorf("unexpected format: %s", d.Header.Format)
	}

	if len(d.Files) != 2 {
		t.Fatalf("expected 2 file paragraphs, got %d", len(d.Files))
	}

	if d.Files[0].License != "MIT" {
		t.Errorf("first paragraph license = %q, want MIT", d.Files[0].License)
	}
	assertSlice(t, d.Files[0].Patterns, []string{"src/*"})

	if d.Files[1].License != "CC0-1.0" {
		t.Errorf("second paragraph license = %q, want CC0-1.0", d.Files[1].License)
	}
}

func TestParseDep5_MultiLineCopyright(t *testing.T) {
	content := `Format: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/

Files: *
Copyright: 2020 Alice
 2021 Bob
 2022 Charlie
License: MIT
`

	d, err := ParseDep5(content)
	if err != nil {
		t.Fatal(err)
	}

	info, ok := d.ReuseInfoOf("anything.go")
	if !ok {
		t.Fatal("expected match")
	}
	assertSlice(t, info.CopyrightNotices, []string{"2020 Alice", "2021 Bob", "2022 Charlie"})
}

func TestParseDep5_ContinuationDot(t *testing.T) {
	content := `Format: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/

Files: *
Copyright: 2020 Alice
 .
 2022 Charlie
License: MIT
`

	d, err := ParseDep5(content)
	if err != nil {
		t.Fatal(err)
	}

	info, ok := d.ReuseInfoOf("test.txt")
	if !ok {
		t.Fatal("expected match")
	}
	assertSlice(t, info.CopyrightNotices, []string{"2020 Alice", "2022 Charlie"})
}

func TestParseDep5_WildcardPatterns(t *testing.T) {
	content := `Format: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/

Files: *.go
Copyright: 2024 Go Author
License: MIT

Files: docs/*
Copyright: 2024 Doc Author
License: CC0-1.0
`

	d, err := ParseDep5(content)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		path    string
		license string
		ok      bool
	}{
		{"main.go", "MIT", true},
		{"sub/main.go", "MIT", true},         // dep5 * matches /
		{"docs/guide.md", "CC0-1.0", true},
		{"docs/sub/deep.md", "CC0-1.0", true}, // dep5 * matches /
		{"readme.txt", "", false},
	}

	for _, tt := range tests {
		info, ok := d.ReuseInfoOf(tt.path)
		if ok != tt.ok {
			t.Errorf("ReuseInfoOf(%q) ok = %v, want %v", tt.path, ok, tt.ok)
			continue
		}
		if ok && info.LicenseExpressions[0] != tt.license {
			t.Errorf("ReuseInfoOf(%q) license = %q, want %q", tt.path, info.LicenseExpressions[0], tt.license)
		}
	}
}

func TestParseDep5_MultiplePatterns(t *testing.T) {
	content := `Format: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/

Files: *.go *.py *.rs
Copyright: 2024 Multi Author
License: Apache-2.0
`

	d, err := ParseDep5(content)
	if err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{"main.go", "script.py", "lib.rs"} {
		info, ok := d.ReuseInfoOf(path)
		if !ok {
			t.Errorf("expected match for %s", path)
			continue
		}
		if info.LicenseExpressions[0] != "Apache-2.0" {
			t.Errorf("license for %s = %q", path, info.LicenseExpressions[0])
		}
	}
}

func TestParseDep5_MissingFields(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			"missing copyright",
			`Format: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/

Files: *
License: MIT
`,
		},
		{
			"missing license",
			`Format: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/

Files: *
Copyright: 2024 Author
`,
		},
		{
			"missing files",
			`Format: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/

Copyright: 2024 Author
License: MIT
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseDep5(tt.content)
			if err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestParseDep5_Empty(t *testing.T) {
	_, err := ParseDep5("")
	if err == nil {
		t.Error("expected error for empty content")
	}
}

func TestParseDep5File(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dep5")
	content := `Format: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/

Files: *
Copyright: 2024 Test
License: MIT
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	d, err := ParseDep5File(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Files) != 1 {
		t.Errorf("expected 1 file paragraph, got %d", len(d.Files))
	}
}

func TestDep5_ReuseInfoOf_NoMatch(t *testing.T) {
	content := `Format: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/

Files: *.go
Copyright: 2024 Author
License: MIT
`

	d, err := ParseDep5(content)
	if err != nil {
		t.Fatal(err)
	}

	_, ok := d.ReuseInfoOf("readme.txt")
	if ok {
		t.Error("expected no match for readme.txt")
	}
}

func TestDep5_ReuseInfoOf_SourceType(t *testing.T) {
	content := `Format: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/

Files: *
Copyright: 2024 Author
License: MIT
`

	d, err := ParseDep5(content)
	if err != nil {
		t.Fatal(err)
	}

	info, ok := d.ReuseInfoOf("test.go")
	if !ok {
		t.Fatal("expected match")
	}
	if info.SourceType != core.Dep5Source {
		t.Errorf("SourceType = %v, want Dep5Source", info.SourceType)
	}
}

func TestDep5_QuestionMark(t *testing.T) {
	content := `Format: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/

Files: src/?.go
Copyright: 2024 Author
License: MIT
`

	d, err := ParseDep5(content)
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := d.ReuseInfoOf("src/a.go"); !ok {
		t.Error("expected match for src/a.go")
	}
	if _, ok := d.ReuseInfoOf("src/ab.go"); ok {
		t.Error("expected no match for src/ab.go")
	}
}

func TestDep5Match(t *testing.T) {
	tests := []struct {
		pattern string
		name    string
		want    bool
	}{
		{"*", "anything", true},
		{"*", "", true},
		{"*.go", "main.go", true},
		{"*.go", "sub/dir/main.go", true},
		{"*.go", "main.c", false},
		{"a*b*c", "aXbYc", true},
		{"a*b*c", "abc", true},
		{"a*b*c", "aXbY", false},
		{"src/?.go", "src/a.go", true},
		{"src/?.go", "src/ab.go", false},
		{"", "", true},
		{"", "x", false},
		{"abc", "abc", true},
		{"abc", "abd", false},
		{"a**b", "axxb", true},
		{"**", "a/b/c", true},
		{"docs/*", "docs/a/b/c.md", true},
	}
	for _, tt := range tests {
		got := dep5Match(tt.pattern, tt.name)
		if got != tt.want {
			t.Errorf("dep5Match(%q, %q) = %v, want %v", tt.pattern, tt.name, got, tt.want)
		}
	}
}

func TestDep5MatchPathological(t *testing.T) {
	// A pattern with many wildcards against a string that almost matches
	// then fails at the end. The previous recursive implementation took
	// exponential time on inputs like this; the iterative version is linear.
	pattern := strings.Repeat("*a", 30) + "*b"
	name := strings.Repeat("a", 100) + "c"

	done := make(chan bool, 1)
	go func() {
		done <- dep5Match(pattern, name)
	}()

	select {
	case got := <-done:
		if got {
			t.Error("expected no match")
		}
	case <-time.After(time.Second):
		t.Fatal("dep5Match took >1s on pathological input")
	}
}

func assertSlice(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("got %v (len %d), want %v (len %d)", got, len(got), want, len(want))
		return
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("index %d: got %q, want %q", i, got[i], want[i])
		}
	}
}
