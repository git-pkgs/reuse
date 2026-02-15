package toml

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/git-pkgs/reuse/internal/core"
)

func TestParseReuseTOML_SingleAnnotation(t *testing.T) {
	content := `version = 1

[[annotations]]
path = "src/**/*.go"
SPDX-FileCopyrightText = "2024 Alice"
SPDX-License-Identifier = "MIT"
`

	rt, err := ParseReuseTOML(content)
	if err != nil {
		t.Fatal(err)
	}

	if rt.Version != 1 {
		t.Errorf("version = %d, want 1", rt.Version)
	}
	if len(rt.Annotations) != 1 {
		t.Fatalf("expected 1 annotation, got %d", len(rt.Annotations))
	}

	ann := rt.Annotations[0]
	assertSlice(t, ann.Paths, []string{"src/**/*.go"})
	assertSlice(t, ann.Copyrights, []string{"2024 Alice"})
	assertSlice(t, ann.Licenses, []string{"MIT"})
	if ann.Precedence != core.Closest {
		t.Errorf("precedence = %v, want Closest", ann.Precedence)
	}
}

func TestParseReuseTOML_MultipleAnnotations(t *testing.T) {
	content := `version = 1

[[annotations]]
path = "src/**"
SPDX-FileCopyrightText = "2024 Alice"
SPDX-License-Identifier = "MIT"

[[annotations]]
path = "docs/**"
precedence = "override"
SPDX-FileCopyrightText = "2024 Bob"
SPDX-License-Identifier = "CC-BY-SA-4.0"
`

	rt, err := ParseReuseTOML(content)
	if err != nil {
		t.Fatal(err)
	}

	if len(rt.Annotations) != 2 {
		t.Fatalf("expected 2 annotations, got %d", len(rt.Annotations))
	}

	if rt.Annotations[1].Precedence != core.Override {
		t.Errorf("second annotation precedence = %v, want Override", rt.Annotations[1].Precedence)
	}
}

func TestParseReuseTOML_ArrayPaths(t *testing.T) {
	content := `version = 1

[[annotations]]
path = ["src/**", "lib/**"]
SPDX-FileCopyrightText = "2024 Multi"
SPDX-License-Identifier = "Apache-2.0"
`

	rt, err := ParseReuseTOML(content)
	if err != nil {
		t.Fatal(err)
	}

	assertSlice(t, rt.Annotations[0].Paths, []string{"src/**", "lib/**"})
}

func TestParseReuseTOML_ArrayCopyrightAndLicense(t *testing.T) {
	content := `version = 1

[[annotations]]
path = "*"
SPDX-FileCopyrightText = ["2024 Alice", "2023 Bob"]
SPDX-License-Identifier = ["MIT", "Apache-2.0"]
`

	rt, err := ParseReuseTOML(content)
	if err != nil {
		t.Fatal(err)
	}

	assertSlice(t, rt.Annotations[0].Copyrights, []string{"2024 Alice", "2023 Bob"})
	assertSlice(t, rt.Annotations[0].Licenses, []string{"MIT", "Apache-2.0"})
}

func TestParseReuseTOML_AllPrecedences(t *testing.T) {
	tests := []struct {
		input string
		want  core.PrecedenceType
	}{
		{`version = 1
[[annotations]]
path = "*"
SPDX-FileCopyrightText = "x"
SPDX-License-Identifier = "MIT"
`, core.Closest},
		{`version = 1
[[annotations]]
path = "*"
precedence = "closest"
SPDX-FileCopyrightText = "x"
SPDX-License-Identifier = "MIT"
`, core.Closest},
		{`version = 1
[[annotations]]
path = "*"
precedence = "aggregate"
SPDX-FileCopyrightText = "x"
SPDX-License-Identifier = "MIT"
`, core.Aggregate},
		{`version = 1
[[annotations]]
path = "*"
precedence = "override"
SPDX-FileCopyrightText = "x"
SPDX-License-Identifier = "MIT"
`, core.Override},
	}

	for _, tt := range tests {
		rt, err := ParseReuseTOML(tt.input)
		if err != nil {
			t.Fatal(err)
		}
		if rt.Annotations[0].Precedence != tt.want {
			t.Errorf("precedence = %v, want %v", rt.Annotations[0].Precedence, tt.want)
		}
	}
}

func TestParseReuseTOML_MissingVersion(t *testing.T) {
	content := `[[annotations]]
path = "*"
SPDX-FileCopyrightText = "x"
SPDX-License-Identifier = "MIT"
`
	_, err := ParseReuseTOML(content)
	if err == nil {
		t.Error("expected error for missing version")
	}
}

func TestParseReuseTOML_WrongVersion(t *testing.T) {
	content := `version = 2
[[annotations]]
path = "*"
SPDX-FileCopyrightText = "x"
SPDX-License-Identifier = "MIT"
`
	_, err := ParseReuseTOML(content)
	if err == nil {
		t.Error("expected error for wrong version")
	}
}

func TestParseReuseTOML_InvalidTOML(t *testing.T) {
	_, err := ParseReuseTOML("not valid [[[toml")
	if err == nil {
		t.Error("expected error for invalid TOML")
	}
}

func TestParseReuseTOML_InvalidPrecedence(t *testing.T) {
	content := `version = 1
[[annotations]]
path = "*"
precedence = "bogus"
SPDX-FileCopyrightText = "x"
SPDX-License-Identifier = "MIT"
`
	_, err := ParseReuseTOML(content)
	if err == nil {
		t.Error("expected error for invalid precedence")
	}
}

func TestParseReuseTOML_MissingPaths(t *testing.T) {
	content := `version = 1
[[annotations]]
SPDX-FileCopyrightText = "x"
SPDX-License-Identifier = "MIT"
`
	_, err := ParseReuseTOML(content)
	if err == nil {
		t.Error("expected error for missing paths")
	}
}

func TestAnnotation_Matches(t *testing.T) {
	ann := Annotation{
		Paths: []string{"src/**/*.go", "lib/*.go"},
	}

	tests := []struct {
		path string
		want bool
	}{
		{"src/main.go", true},
		{"src/sub/util.go", true},
		{"lib/helper.go", true},
		{"lib/sub/deep.go", false},
		{"other/file.go", false},
	}

	for _, tt := range tests {
		if got := ann.Matches(tt.path); got != tt.want {
			t.Errorf("Matches(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestReuseTOML_ReuseInfoOf_LastMatchWins(t *testing.T) {
	content := `version = 1

[[annotations]]
path = "**"
SPDX-FileCopyrightText = "2024 First"
SPDX-License-Identifier = "MIT"

[[annotations]]
path = "src/**"
SPDX-FileCopyrightText = "2024 Second"
SPDX-License-Identifier = "Apache-2.0"
`

	rt, err := ParseReuseTOML(content)
	if err != nil {
		t.Fatal(err)
	}

	// src/main.go matches both; last match (Apache-2.0) wins.
	info, prec, ok := rt.ReuseInfoOf("src/main.go")
	if !ok {
		t.Fatal("expected match")
	}
	assertSlice(t, info.LicenseExpressions, []string{"Apache-2.0"})
	assertSlice(t, info.CopyrightNotices, []string{"2024 Second"})
	if prec != core.Closest {
		t.Errorf("precedence = %v, want Closest", prec)
	}

	// README.md only matches first.
	info, _, ok = rt.ReuseInfoOf("README.md")
	if !ok {
		t.Fatal("expected match")
	}
	assertSlice(t, info.LicenseExpressions, []string{"MIT"})
}

func TestReuseTOML_ReuseInfoOf_NoMatch(t *testing.T) {
	content := `version = 1

[[annotations]]
path = "src/**"
SPDX-FileCopyrightText = "2024 Author"
SPDX-License-Identifier = "MIT"
`

	rt, err := ParseReuseTOML(content)
	if err != nil {
		t.Fatal(err)
	}

	_, _, ok := rt.ReuseInfoOf("other/file.go")
	if ok {
		t.Error("expected no match")
	}
}

func TestReuseTOML_ReuseInfoOf_SourceType(t *testing.T) {
	content := `version = 1

[[annotations]]
path = "**"
SPDX-FileCopyrightText = "2024 Author"
SPDX-License-Identifier = "MIT"
`

	rt, err := ParseReuseTOML(content)
	if err != nil {
		t.Fatal(err)
	}

	info, _, ok := rt.ReuseInfoOf("test.go")
	if !ok {
		t.Fatal("expected match")
	}
	if info.SourceType != core.ReuseToml {
		t.Errorf("SourceType = %v, want ReuseToml", info.SourceType)
	}
}

func TestParseReuseTOMLFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "REUSE.toml")
	content := `version = 1

[[annotations]]
path = "**"
SPDX-FileCopyrightText = "2024 Test"
SPDX-License-Identifier = "MIT"
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	rt, err := ParseReuseTOMLFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if rt.Source != path {
		t.Errorf("Source = %q, want %q", rt.Source, path)
	}
	if len(rt.Annotations) != 1 {
		t.Errorf("expected 1 annotation, got %d", len(rt.Annotations))
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
