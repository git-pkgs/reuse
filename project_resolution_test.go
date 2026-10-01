package reuse

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestProjectSidecarPrecedence(t *testing.T) {
	for _, tc := range []struct {
		precedence string
		want       []string
	}{
		{"closest", []string{"MIT"}},
		{"aggregate", []string{"Apache-2.0", "MIT"}},
		{"override", []string{"Apache-2.0"}},
	} {
		t.Run(tc.precedence, func(t *testing.T) {
			root := t.TempDir()
			mkfile(t, root, "file.go", "// SPDX-License-Identifier: GPL-3.0-only\n")
			mkfile(t, root, "file.go.license", "SPDX-License-Identifier: MIT\n")
			mkfile(t, root, "REUSE.toml", annotation("*.go", tc.precedence, "Apache-2.0", "2024 Annotation Author"))
			project, err := OpenProject(root)
			if err != nil {
				t.Fatal(err)
			}
			info, err := project.ReuseInfoOf("file.go")
			if err != nil {
				t.Fatal(err)
			}
			slices.Sort(info.LicenseExpressions)
			assertSlice(t, info.LicenseExpressions, tc.want)
			assertSlice(t, info.CopyrightNotices, []string{"2024 Annotation Author"})
		})
	}
}

func TestProjectNestedAnnotations(t *testing.T) {
	for _, tc := range []struct {
		name, outer, inner, header string
		want                       []string
	}{
		{"closest", "closest", "closest", "", []string{"Apache-2.0"}},
		{"header", "closest", "closest", "GPL-3.0-only", []string{"GPL-3.0-only"}},
		{"outer aggregate", "aggregate", "closest", "", []string{"Apache-2.0", "MIT"}},
		{"aggregate with header", "aggregate", "closest", "GPL-3.0-only", []string{"GPL-3.0-only", "MIT"}},
		{"inner aggregate", "closest", "aggregate", "", []string{"Apache-2.0", "MIT"}},
		{"inner override", "closest", "override", "GPL-3.0-only", []string{"Apache-2.0"}},
		{"aggregate and override", "aggregate", "override", "GPL-3.0-only", []string{"Apache-2.0", "MIT"}},
		{"outer override", "override", "closest", "GPL-3.0-only", []string{"MIT"}},
		{"two overrides", "override", "override", "GPL-3.0-only", []string{"MIT"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			mkfile(t, root, "REUSE.toml", annotation("src/**", tc.outer, "MIT", ""))
			mkfile(t, root, "src/REUSE.toml", annotation("*.go", tc.inner, "Apache-2.0", ""))
			mkfile(t, root, "src/main.go", "// SPDX-License-Identifier: "+tc.header+"\n")
			project, err := OpenProject(root)
			if err != nil {
				t.Fatal(err)
			}
			info, err := project.ReuseInfoOf(filepath.Join("src", "main.go"))
			if err != nil {
				t.Fatal(err)
			}
			slices.Sort(info.LicenseExpressions)
			assertSlice(t, info.LicenseExpressions, tc.want)
		})
	}
}

func TestProjectNestedOnlyAndRelativePaths(t *testing.T) {
	root := t.TempDir()
	mkfile(t, root, "src/REUSE.toml", annotation("*.go", "closest", "MIT", "2024 Nested Author"))
	for _, path := range []string{"src/main.go", "src/sub/main.go", "other/main.go"} {
		mkfile(t, root, path, "package main\n")
	}
	project, err := OpenProject(root)
	if err != nil {
		t.Fatal(err)
	}
	all, err := project.AllReuseInfo()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 {
		t.Fatalf("covered files = %d, want 3", len(all))
	}
	for path, info := range all {
		if filepath.ToSlash(path) == "src/main.go" {
			assertSlice(t, info.LicenseExpressions, []string{"MIT"})
			if info.SourcePath != "src/REUSE.toml" {
				t.Fatalf("source = %q", info.SourcePath)
			}
		} else if !info.IsEmpty() {
			t.Fatalf("annotation escaped its scope: %s: %+v", path, info)
		}
	}
}

func TestProjectNestedClosestFillsFieldsSeparately(t *testing.T) {
	root := t.TempDir()
	mkfile(t, root, "REUSE.toml", annotation("**", "closest", "MIT", "2024 Outer Author"))
	mkfile(t, root, "src/REUSE.toml", annotation("**", "closest", "", "2024 Inner Author"))
	mkfile(t, root, "src/sub/main.go", "package main\n")
	project, err := OpenProject(root)
	if err != nil {
		t.Fatal(err)
	}
	info, err := project.ReuseInfoOf(filepath.Join("src", "sub", "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	assertSlice(t, info.LicenseExpressions, []string{"MIT"})
	assertSlice(t, info.CopyrightNotices, []string{"2024 Inner Author"})
}

func TestProjectSidecarDep5Aggregate(t *testing.T) {
	root := setupFakeProject(t, "dep5")
	mkfile(t, root, "src/main.go.license", "SPDX-License-Identifier: Apache-2.0\n")
	project, err := OpenProject(root)
	if err != nil {
		t.Fatal(err)
	}
	info, err := project.ReuseInfoOf(filepath.Join("src", "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	assertSlice(t, info.LicenseExpressions, []string{"Apache-2.0", "MIT"})
}

func TestProjectResultStorage(t *testing.T) {
	root := t.TempDir()
	mkfile(t, root, "REUSE.toml", annotation("*.go", "closest", "MIT", "2024 Author"))
	mkfile(t, root, "main.go", "// SPDX-FileContributor: Contributor\n")
	project, err := OpenProject(root)
	if err != nil {
		t.Fatal(err)
	}
	for range 8 {
		t.Run("worker", func(t *testing.T) {
			t.Parallel()
			for range 10 {
				info, err := project.ReuseInfoOf("main.go")
				if err != nil {
					t.Fatal(err)
				}
				assertSlice(t, info.LicenseExpressions, []string{"MIT"})
				assertSlice(t, info.CopyrightNotices, []string{"2024 Author"})
				assertSlice(t, info.Contributors, []string{"Contributor"})
				info.LicenseExpressions[0] = "changed"
				info.CopyrightNotices[0] = "changed"
			}
		})
	}
}

func TestProjectRejectsNonlocalPaths(t *testing.T) {
	project, err := OpenProject(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"", "../file.go", filepath.Join(project.Root, "file.go")} {
		if _, err := project.ReuseInfoOf(path); err == nil {
			t.Errorf("accepted nonlocal path %q", path)
		}
	}
}

func TestOpenProjectNestedMetadata(t *testing.T) {
	root := t.TempDir()
	mkfile(t, root, ".git/REUSE.toml", "invalid TOML")
	mkfile(t, root, "LICENSES/REUSE.toml", "invalid TOML")
	if _, err := OpenProject(root); err != nil {
		t.Fatal(err)
	}
	mkfile(t, root, "src/REUSE.toml", "invalid TOML")
	if _, err := OpenProject(root); err == nil {
		t.Fatal("accepted malformed nested annotations")
	}
}

func TestOpenProjectSymlinkRoot(t *testing.T) {
	root := t.TempDir()
	mkfile(t, root, "REUSE.toml", annotation("src/**", "closest", "MIT", ""))
	mkfile(t, root, "src/main.go", "package main\n")
	link := filepath.Join(t.TempDir(), "project")
	if err := os.Symlink(root, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	project, err := OpenProject(link)
	if err != nil {
		t.Fatal(err)
	}
	info, err := project.ReuseInfoOf(filepath.Join("src", "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	assertSlice(t, info.LicenseExpressions, []string{"MIT"})
}

func annotation(path, precedence, license, copyright string) string {
	text := fmt.Sprintf("version = 1\n[[annotations]]\npath = %q\nprecedence = %q\n", path, precedence)
	if license != "" {
		text += fmt.Sprintf("SPDX-License-Identifier = %q\n", license)
	}
	if copyright != "" {
		text += fmt.Sprintf("SPDX-FileCopyrightText = %q\n", copyright)
	}
	return text
}
