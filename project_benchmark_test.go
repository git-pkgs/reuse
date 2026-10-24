package reuse_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/git-pkgs/reuse"
)

func BenchmarkProject(b *testing.B) {
	root := b.TempDir()
	text := "// SPDX-License-Identifier: MIT\n" + strings.Repeat("// source line\n", 4096)
	for i := range 16 {
		if err := os.WriteFile(filepath.Join(root, fmt.Sprintf("file%d.go", i)), []byte(text), 0600); err != nil {
			b.Fatal(err)
		}
	}
	config := "version = 1\n[[annotations]]\npath = \"*.go\"\nSPDX-FileCopyrightText = \"2024 Example\"\n"
	if err := os.WriteFile(filepath.Join(root, "REUSE.toml"), []byte(config), 0600); err != nil {
		b.Fatal(err)
	}
	project, err := reuse.OpenProject(root)
	if err != nil {
		b.Fatal(err)
	}
	b.Run("ReuseInfoOf", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			info, err := project.ReuseInfoOf("file0.go")
			if err != nil || !info.HasLicense() {
				b.Fatalf("info=%+v err=%v", info, err)
			}
		}
	})
	b.Run("AllReuseInfo", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			info, err := project.AllReuseInfo()
			if err != nil || len(info) != 16 {
				b.Fatalf("files=%d err=%v", len(info), err)
			}
		}
	})
}
