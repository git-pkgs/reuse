package reuse_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/git-pkgs/reuse"
)

func TestProjectUTF8ProbeBoundary(t *testing.T) {
	const boundary = 8192
	const header = "// SPDX-License-Identifier: MIT\n"
	for _, character := range []string{"é", "€", "\U00010400"} {
		for split := 1; split < len(character); split++ {
			text := header + strings.Repeat(" ", boundary-split-len(header)) + character + "\n"
			t.Run(character+strings.Repeat("_", split), func(t *testing.T) {
				checkProjectLicense(t, text, []string{"MIT"})
			})
		}
	}
	for _, tc := range []struct {
		name string
		text string
	}{
		{"invalid-before-boundary", header + "\xff" + strings.Repeat(" ", boundary)},
		{"invalid-at-boundary", header + strings.Repeat(" ", boundary-1-len(header)) + "\xc3x"},
		{"incomplete-at-EOF", header + strings.Repeat(" ", boundary-1-len(header)) + "\xe2\x82"},
		{"NUL", header + "\x00" + strings.Repeat(" ", boundary)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			checkProjectLicense(t, tc.text, nil)
		})
	}
}

func checkProjectLicense(t *testing.T, text string, want []string) {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "source.go"), []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	project, err := reuse.OpenProject(root)
	if err != nil {
		t.Fatal(err)
	}
	info, err := project.ReuseInfoOf("source.go")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(info.LicenseExpressions, want) {
		t.Fatalf("licenses = %q, want %q", info.LicenseExpressions, want)
	}
}
