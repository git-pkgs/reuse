package reuse_test

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/git-pkgs/reuse"
)

func TestProjectFSMatchesDirectory(t *testing.T) {
	for _, mode := range []string{"headers", "toml", "nested-only", "dep5"} {
		t.Run(mode, func(t *testing.T) {
			files := projectFiles(mode)
			root := writeProjectFiles(t, files)
			directory, err := reuse.OpenProject(root)
			if err != nil {
				t.Fatal(err)
			}
			memory, err := reuse.OpenProjectFS(files)
			if err != nil {
				t.Fatal(err)
			}
			if memory.Root != "." || !filepath.IsAbs(directory.Root) {
				t.Fatalf("roots: memory = %q, directory = %q", memory.Root, directory.Root)
			}
			if !reflect.DeepEqual(memory.ReuseTOMLs, directory.ReuseTOMLs) ||
				!reflect.DeepEqual(memory.Dep5, directory.Dep5) ||
				!reflect.DeepEqual(memory.LicenseFiles, directory.LicenseFiles) {
				t.Fatalf("metadata differs: memory = %+v, directory = %+v", memory, directory)
			}
			if !slices.Equal(memory.LicenseFiles, []string{"LICENSES/MIT.txt"}) {
				t.Fatalf("license files = %q", memory.LicenseFiles)
			}
			got, err := memory.AllReuseInfo()
			if err != nil {
				t.Fatal(err)
			}
			want, err := directory.AllReuseInfo()
			if err != nil {
				t.Fatal(err)
			}
			checkProjectFSParity(t, got, want)
			checkProjectFSResults(t, mode, got)
			checkIndividualLookups(t, memory, got)
			checkCoveredFilesFS(t, files, root)
		})
	}
}

func checkProjectFSParity(t *testing.T, got, directory map[string]reuse.ReuseInfo) {
	t.Helper()
	want := make(map[string]reuse.ReuseInfo, len(directory))
	for name, info := range directory {
		want[filepath.ToSlash(name)] = info
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("memory = %+v, directory = %+v", got, want)
	}
}

func checkIndividualLookups(t *testing.T, project *reuse.Project, all map[string]reuse.ReuseInfo) {
	t.Helper()
	for name, want := range all {
		info, err := project.ReuseInfoOf(name)
		if err != nil || !reflect.DeepEqual(info, want) {
			t.Errorf("ReuseInfoOf(%q) = %+v, %v; want %+v", name, info, err, want)
		}
	}
}

func checkCoveredFilesFS(t *testing.T, files fstest.MapFS, root string) {
	t.Helper()
	covered, err := reuse.CoveredFilesFS(files)
	if err != nil {
		t.Fatal(err)
	}
	diskCovered, err := reuse.CoveredFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	for i := range diskCovered {
		diskCovered[i] = filepath.ToSlash(diskCovered[i])
	}
	if !slices.Equal(covered, diskCovered) {
		t.Fatalf("covered files: memory = %q, directory = %q", covered, diskCovered)
	}
}

func projectFiles(mode string) fstest.MapFS {
	files := fstest.MapFS{}
	for name, text := range map[string]string{
		"src/header.go":                "// SPDX-License-Identifier: MIT\n// SPDX-FileCopyrightText: 2024 Header Author\n",
		"src/sidecar.go":               "// SPDX-License-Identifier: GPL-3.0-only\n",
		"src/sidecar.go.license":       "SPDX-License-Identifier: CC0-1.0\nSPDX-FileCopyrightText: 2024 Sidecar Author\n",
		"src/bare.go":                  "package src\n",
		"src/deep/override.go":         "// SPDX-License-Identifier: GPL-3.0-only\n",
		"src/deep/override.go.license": "SPDX-License-Identifier: CC0-1.0\n",
		"src/partial.go":               "// SPDX-FileCopyrightText: 2024 Partial Author\n",
		"binary.bin":                   "SPDX-License-Identifier: MIT\n\x00",
		"invalid.bin":                  "SPDX-License-Identifier: MIT\n\xff",
		"unicode.txt":                  "SPDX-License-Identifier: MIT\n" + strings.Repeat(" ", 8192-1-len("SPDX-License-Identifier: MIT\n")) + "é\n",
		"empty.txt":                    "",
		"uncovered.txt":                "no metadata\n",
		"LICENSE":                      "MIT license text",
		"LICENSES/MIT.txt":             "MIT license text",
		"LICENSES/MIT.txt.license":     "SPDX-License-Identifier: CC0-1.0\n",
		"LICENSES/nested/other.txt":    "license text",
		"LICENSES/nested/REUSE.toml":   "invalid",
		".git/REUSE.toml":              "invalid",
		".hg/REUSE.toml":               "invalid",
		".sl/REUSE.toml":               "invalid",
		".reuse/REUSE.toml":            "invalid",
		"report.spdx.json":             "{}",
	} {
		files[name] = &fstest.MapFile{Data: []byte(text)}
	}
	if mode != "headers" {
		files[".reuse/dep5"] = &fstest.MapFile{Data: []byte("Format: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/\n\nFiles: src/*\nCopyright: 2024 DEP5 Author\nLicense: BSD-2-Clause\n")}
	}
	if mode == "toml" {
		files["REUSE.toml"] = &fstest.MapFile{Data: []byte(fsAnnotation("src/**", "aggregate", "BSD-3-Clause", "2024 Outer Author"))}
	}
	if mode == "toml" || mode == "nested-only" {
		files["src/REUSE.toml"] = &fstest.MapFile{Data: []byte(fsAnnotation("**", "closest", "Apache-2.0", "2024 Inner Author"))}
		files["src/deep/REUSE.toml"] = &fstest.MapFile{Data: []byte(fsAnnotation("*.go", "override", "ISC", "2024 Override Author"))}
	}
	return files
}

func fsAnnotation(pattern, precedence, license, copyright string) string {
	return "version = 1\n[[annotations]]\npath = \"" + pattern + "\"\nprecedence = \"" + precedence + "\"\nSPDX-License-Identifier = \"" + license + "\"\nSPDX-FileCopyrightText = \"" + copyright + "\"\n"
}

func writeProjectFiles(t *testing.T, files fstest.MapFS) string {
	t.Helper()
	root := t.TempDir()
	for name, file := range files {
		filename := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filename, file.Data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func checkProjectFSResults(t *testing.T, mode string, got map[string]reuse.ReuseInfo) {
	t.Helper()
	if len(got) != 9 {
		t.Fatalf("covered files = %d, want 9: %+v", len(got), got)
	}
	want := map[string][]string{
		"src/header.go":        {"MIT"},
		"src/sidecar.go":       {"CC0-1.0"},
		"src/bare.go":          nil,
		"src/partial.go":       nil,
		"src/deep/override.go": {"CC0-1.0"},
		"unicode.txt":          {"MIT"},
		"binary.bin":           nil,
		"invalid.bin":          nil,
		"uncovered.txt":        nil,
	}
	if mode == "toml" || mode == "nested-only" {
		want["src/bare.go"] = []string{"Apache-2.0"}
		want["src/partial.go"] = []string{"Apache-2.0"}
		want["src/deep/override.go"] = []string{"ISC"}
	}
	for name, licenses := range want {
		switch mode {
		case "toml":
			if strings.HasPrefix(name, "src/") {
				licenses = append(licenses, "BSD-3-Clause")
			}
		case "dep5":
			if strings.HasPrefix(name, "src/") {
				licenses = append(licenses, "BSD-2-Clause")
			}
		}
		info, ok := got[name]
		if !ok || !slices.Equal(info.LicenseExpressions, licenses) {
			t.Errorf("%s licenses = %q, want %q (present: %v)", name, info.LicenseExpressions, licenses, ok)
		}
	}
	if got["src/sidecar.go"].SourcePath != "src/sidecar.go.license" || got["src/sidecar.go"].SourceType != reuse.DotLicense {
		t.Errorf("sidecar source = %+v", got["src/sidecar.go"])
	}
	if !slices.Contains(got["src/partial.go"].CopyrightNotices, "2024 Partial Author") {
		t.Errorf("partial copyright = %q", got["src/partial.go"].CopyrightNotices)
	}
}

func TestProjectFSNonRegularEntries(t *testing.T) {
	files := fstest.MapFS{
		"main.go":         {Data: []byte("// SPDX-License-Identifier: MIT\n")},
		"main.go.license": {Mode: fs.ModeSymlink, Data: []byte("missing")},
		"REUSE.toml":      {Mode: fs.ModeSymlink, Data: []byte("missing")},
		"link.go":         {Mode: fs.ModeSymlink, Data: []byte("main.go")},
		"fifo":            {Mode: fs.ModeNamedPipe, Data: []byte("ignored")},
		"socket":          {Mode: fs.ModeSocket, Data: []byte("ignored")},
		"device":          {Mode: fs.ModeDevice, Data: []byte("ignored")},
	}
	project, err := reuse.OpenProjectFS(files)
	if err != nil {
		t.Fatal(err)
	}
	all, err := project.AllReuseInfo()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 || !slices.Equal(all["main.go"].LicenseExpressions, []string{"MIT"}) {
		t.Fatalf("results = %+v", all)
	}
}

func TestProjectFSSubdirectory(t *testing.T) {
	files := fstest.MapFS{
		"snapshot/main.go": {Data: []byte("// SPDX-License-Identifier: MIT\n")},
		"outside.go":       {Data: []byte("// SPDX-License-Identifier: ISC\n")},
	}
	sub, err := fs.Sub(files, "snapshot")
	if err != nil {
		t.Fatal(err)
	}
	project, err := reuse.OpenProjectFS(sub)
	if err != nil {
		t.Fatal(err)
	}
	all, err := project.AllReuseInfo()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 || !slices.Equal(all["main.go"].LicenseExpressions, []string{"MIT"}) {
		t.Fatalf("results = %+v", all)
	}
	for _, name := range []string{"", "../outside.go", "/outside.go", "./main.go", "sub/../main.go", "sub//main.go"} {
		if _, err := project.ReuseInfoOf(name); err == nil {
			t.Errorf("accepted nonlocal path %q", name)
		}
	}
}

func TestProjectFSErrors(t *testing.T) {
	for _, name := range []string{"REUSE.toml", "nested/REUSE.toml", ".reuse/dep5"} {
		t.Run(name, func(t *testing.T) {
			if _, err := reuse.OpenProjectFS(fstest.MapFS{name: {Data: []byte{}}}); err == nil {
				t.Fatal("accepted malformed metadata")
			}
		})
	}
	if _, err := reuse.OpenProjectFS(fstest.MapFS{".": {Data: []byte("file")}}); err == nil {
		t.Fatal("accepted a regular file as root")
	}
	files := fstest.MapFS{"main.go": {Data: []byte("// SPDX-License-Identifier: MIT\n")}}
	project, err := reuse.OpenProjectFS(files)
	if err != nil {
		t.Fatal(err)
	}
	files["main.go.license"] = &fstest.MapFile{Mode: fs.ModeDir}
	if _, err := project.ReuseInfoOf("main.go"); err != nil {
		t.Fatalf("directory sidecar: %v", err)
	}
	if _, err := reuse.OpenProjectFS(failingFS{}); !errors.Is(err, fs.ErrPermission) {
		t.Fatalf("root error = %v, want permission denied", err)
	}
	if _, err := reuse.CoveredFilesFS(failingFS{}); !errors.Is(err, fs.ErrPermission) {
		t.Fatalf("walk error = %v, want permission denied", err)
	}
}

func TestProjectFSNames(t *testing.T) {
	files := fstest.MapFS{
		`back\slash.go`: {Data: []byte("SPDX-License-Identifier: MIT\n")},
		"drive:name.go": {Data: []byte("SPDX-License-Identifier: ISC\n")},
	}
	project, err := reuse.OpenProjectFS(files)
	if err != nil {
		t.Fatal(err)
	}
	all, err := project.AllReuseInfo()
	if err != nil {
		t.Fatal(err)
	}
	for name, license := range map[string]string{`back\slash.go`: "MIT", "drive:name.go": "ISC"} {
		info := all[name]
		if !slices.Equal(info.LicenseExpressions, []string{license}) || info.SourcePath != name {
			t.Errorf("%q = %+v", name, info)
		}
	}
}

func TestProjectFSReadErrors(t *testing.T) {
	for _, name := range []string{"REUSE.toml", "nested/REUSE.toml", ".reuse/dep5"} {
		t.Run(name, func(t *testing.T) {
			files := readErrorFS{FS: fstest.MapFS{name: {Data: []byte("metadata")}}, name: name}
			if _, err := reuse.OpenProjectFS(files); !errors.Is(err, fs.ErrPermission) {
				t.Fatalf("metadata error = %v, want permission denied", err)
			}
		})
	}
	for _, name := range []string{"main.go", "main.go.license"} {
		t.Run(name, func(t *testing.T) {
			files := readErrorFS{FS: fstest.MapFS{
				"main.go": {Data: []byte("package main\n")},
				name:      {Data: []byte("SPDX-License-Identifier: MIT\n")},
			}, name: name}
			project, err := reuse.OpenProjectFS(files)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := project.AllReuseInfo(); !errors.Is(err, fs.ErrPermission) {
				t.Fatalf("file error = %v, want permission denied", err)
			}
		})
	}
}

func TestProjectLiteralRoot(t *testing.T) {
	root := writeProjectFiles(t, fstest.MapFS{"main.go": {Data: []byte("SPDX-License-Identifier: MIT\n")}})
	project := &reuse.Project{Root: root}
	all, err := project.AllReuseInfo()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(all["main.go"].LicenseExpressions, []string{"MIT"}) {
		t.Fatalf("results = %+v", all)
	}
}

func TestProjectFSWrappedNotExist(t *testing.T) {
	for _, name := range []string{"main.go.license", "missing.go"} {
		t.Run(name, func(t *testing.T) {
			files := readErrorFS{
				FS:   fstest.MapFS{"main.go": {Data: []byte("SPDX-License-Identifier: MIT\n")}},
				name: name,
				err:  fmt.Errorf("lookup: %w", fs.ErrNotExist),
			}
			project, err := reuse.OpenProjectFS(files)
			if err != nil {
				t.Fatal(err)
			}
			info, err := project.ReuseInfoOf("main.go")
			if err != nil || !slices.Equal(info.LicenseExpressions, []string{"MIT"}) {
				t.Fatalf("header = %+v, %v", info, err)
			}
			if _, err := project.ReuseInfoOf("missing.go"); err != nil {
				t.Fatalf("missing file: %v", err)
			}
		})
	}
}

type readErrorFS struct {
	fs.FS
	name string
	err  error
}

func (f readErrorFS) Open(name string) (fs.File, error) {
	if name == f.name {
		if f.err != nil {
			return nil, f.err
		}
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrPermission}
	}
	return f.FS.Open(name)
}

func (f readErrorFS) Stat(name string) (fs.FileInfo, error) {
	if name == f.name && f.err != nil {
		return nil, f.err
	}
	return fs.Stat(f.FS, name)
}

type failingFS struct{}

func (failingFS) Open(name string) (fs.File, error) {
	return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrPermission}
}
