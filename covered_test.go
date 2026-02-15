package reuse

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestIsIgnoredDir(t *testing.T) {
	ignored := []string{".git", ".hg", ".sl", "LICENSES", ".reuse"}
	for _, name := range ignored {
		if !IsIgnoredDir(name) {
			t.Errorf("expected %q to be ignored", name)
		}
	}

	notIgnored := []string{"src", "docs", "licenses", "git", "reuse"}
	for _, name := range notIgnored {
		if IsIgnoredDir(name) {
			t.Errorf("expected %q to not be ignored", name)
		}
	}
}

func TestIsIgnoredFile(t *testing.T) {
	ignored := []string{
		"LICENSE", "LICENSE.txt", "LICENSE-MIT", "LICENSE.md",
		"LICENCE", "LICENCE.txt",
		"COPYING", "COPYING.txt", "COPYING-GPL",
		"foo.license", "image.png.license",
		"REUSE.toml",
		"data.spdx", "data.spdx.json", "data.spdx.rdf",
		"data.spdx.xml", "data.spdx.yml", "data.spdx.yaml",
		".git",
		".hgtags",
	}
	for _, name := range ignored {
		if !IsIgnoredFile(name) {
			t.Errorf("expected %q to be ignored", name)
		}
	}

	notIgnored := []string{
		"main.go", "README.md", "Makefile",
		"license_check.go", // contains "license" but not the LICENSE pattern
		"COPYING_test.go",  // doesn't match COPYING[-.]
	}
	for _, name := range notIgnored {
		if IsIgnoredFile(name) {
			t.Errorf("expected %q to not be ignored", name)
		}
	}
}

func TestIsCoveredFile(t *testing.T) {
	covered := []string{
		"main.go",
		"src/util.go",
		"docs/guide.md",
		"README.md",
	}
	for _, path := range covered {
		if !IsCoveredFile(path) {
			t.Errorf("expected %q to be covered", path)
		}
	}

	notCovered := []string{
		"LICENSE",
		"LICENSE.txt",
		"COPYING",
		".git/config",
		"LICENSES/MIT.txt",
		".reuse/dep5",
		"REUSE.toml",
		"foo.license",
		"data.spdx.json",
	}
	for _, path := range notCovered {
		if IsCoveredFile(path) {
			t.Errorf("expected %q to not be covered", path)
		}
	}
}

func TestCoveredFiles(t *testing.T) {
	root := t.TempDir()

	// Create covered files.
	mkfile(t, root, "main.go", "package main")
	mkfile(t, root, "src/util.go", "package src")
	mkfile(t, root, "README.md", "# Hello")

	// Create excluded files.
	mkfile(t, root, "LICENSE", "MIT License")
	mkfile(t, root, "COPYING", "GPL")
	mkfile(t, root, "REUSE.toml", "version = 1")
	mkfile(t, root, "image.png.license", "SPDX...")
	mkfile(t, root, "LICENSES/MIT.txt", "MIT text")
	mkfile(t, root, ".reuse/dep5", "Format: ...")
	mkfile(t, root, ".git/config", "[core]")
	mkfile(t, root, "data.spdx.json", "{}")

	// Create zero-byte file.
	mkfile(t, root, "empty.txt", "")

	files, err := CoveredFiles(root)
	if err != nil {
		t.Fatal(err)
	}

	sort.Strings(files)
	expected := []string{"README.md", "main.go", "src/util.go"}
	assertSlice(t, files, expected)
}

func TestCoveredFiles_NoSymlinks(t *testing.T) {
	root := t.TempDir()

	mkfile(t, root, "real.go", "package main")
	target := filepath.Join(root, "real.go")
	link := filepath.Join(root, "link.go")
	if err := os.Symlink(target, link); err != nil {
		t.Skip("symlinks not supported")
	}

	files, err := CoveredFiles(root)
	if err != nil {
		t.Fatal(err)
	}

	// Only real.go should appear, not link.go.
	assertSlice(t, files, []string{"real.go"})
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

func mkfile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
