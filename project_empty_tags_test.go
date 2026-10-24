package reuse_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/git-pkgs/reuse"
)

func TestProjectEmptyTagsAllowAnnotationFallback(t *testing.T) {
	for _, sidecar := range []bool{false, true} {
		name := "header"
		if sidecar {
			name = "sidecar"
		}
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			files := map[string]string{
				"REUSE.toml": "version = 1\n[[annotations]]\npath = \"*.go\"\n" +
					"SPDX-License-Identifier = \"MIT\"\nSPDX-FileCopyrightText = \"2024 Author\"\n",
				"main.go": "// SPDX-License-Identifier: \t\n// SPDX-FileCopyrightText:\n" +
					"// SPDX-FileContributor:\npackage main\n",
			}
			if sidecar {
				files["main.go.license"] = "SPDX-License-Identifier:\n\nSPDX-FileCopyrightText:\n" +
					"SPDX-FileContributor:\nSPDX-FileContributor: Contributor\n"
			}
			for path, text := range files {
				if err := os.WriteFile(filepath.Join(root, path), []byte(text), 0600); err != nil {
					t.Fatal(err)
				}
			}
			project, err := reuse.OpenProject(root)
			if err != nil {
				t.Fatal(err)
			}
			info, err := project.ReuseInfoOf("main.go")
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(info.LicenseExpressions, []string{"MIT"}) ||
				!slices.Equal(info.CopyrightNotices, []string{"2024 Author"}) {
				t.Fatalf("annotation fallback: %+v", info)
			}
			var contributors []string
			if sidecar {
				contributors = []string{"Contributor"}
			}
			if !slices.Equal(info.Contributors, contributors) {
				t.Fatalf("contributors = %q, want %q", info.Contributors, contributors)
			}
		})
	}
}
