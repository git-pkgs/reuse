package reuse

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenProject_WithReuseTOML(t *testing.T) {
	root := setupFakeProject(t, "toml")

	p, err := OpenProject(root)
	if err != nil {
		t.Fatal(err)
	}

	if p.ReuseTOML == nil {
		t.Fatal("expected ReuseTOML to be set")
	}
	if p.Dep5 != nil {
		t.Error("expected Dep5 to be nil when REUSE.toml exists")
	}
	if len(p.LicenseFiles) == 0 {
		t.Error("expected license files in LICENSES/")
	}
}

func TestOpenProject_WithDep5(t *testing.T) {
	root := setupFakeProject(t, "dep5")

	p, err := OpenProject(root)
	if err != nil {
		t.Fatal(err)
	}

	if p.ReuseTOML != nil {
		t.Error("expected ReuseTOML to be nil")
	}
	if p.Dep5 == nil {
		t.Fatal("expected Dep5 to be set")
	}
}

func TestOpenProject_HeaderOnly(t *testing.T) {
	root := setupFakeProject(t, "header")

	p, err := OpenProject(root)
	if err != nil {
		t.Fatal(err)
	}

	if p.ReuseTOML != nil {
		t.Error("expected ReuseTOML to be nil")
	}
	if p.Dep5 != nil {
		t.Error("expected Dep5 to be nil")
	}
}

func TestProject_ReuseInfoOf_FileHeader(t *testing.T) {
	root := setupFakeProject(t, "header")

	p, err := OpenProject(root)
	if err != nil {
		t.Fatal(err)
	}

	info, err := p.ReuseInfoOf("src/main.go")
	if err != nil {
		t.Fatal(err)
	}

	assertSlice(t, info.LicenseExpressions, []string{"MIT"})
	assertSlice(t, info.CopyrightNotices, []string{"2024 Test Author"})
}

func TestProject_ReuseInfoOf_DotLicenseSidecar(t *testing.T) {
	root := setupFakeProject(t, "header")

	// Add a .license sidecar for an image.
	mkfile(t, root, "assets/logo.png", "\x89PNG binary")
	mkfile(t, root, "assets/logo.png.license", `SPDX-License-Identifier: CC0-1.0
SPDX-FileCopyrightText: 2024 Designer`)

	p, err := OpenProject(root)
	if err != nil {
		t.Fatal(err)
	}

	info, err := p.ReuseInfoOf("assets/logo.png")
	if err != nil {
		t.Fatal(err)
	}

	assertSlice(t, info.LicenseExpressions, []string{"CC0-1.0"})
	assertSlice(t, info.CopyrightNotices, []string{"2024 Designer"})
	if info.SourceType != DotLicense {
		t.Errorf("SourceType = %v, want DotLicense", info.SourceType)
	}
}

func TestProject_ReuseInfoOf_Override(t *testing.T) {
	root := setupFakeProject(t, "toml")

	// The file has MIT in its header, but REUSE.toml overrides it.
	mkfile(t, root, "docs/guide.md", `<!-- SPDX-License-Identifier: MIT -->
<!-- SPDX-FileCopyrightText: 2024 File Author -->`)

	// Set up an override annotation for docs/.
	writeReuseTOML(t, root, `version = 1

[[annotations]]
path = "src/**"
SPDX-FileCopyrightText = "2024 TOML Author"
SPDX-License-Identifier = "MIT"

[[annotations]]
path = "docs/**"
precedence = "override"
SPDX-FileCopyrightText = "2024 Override Author"
SPDX-License-Identifier = "CC-BY-SA-4.0"
`)

	p, err := OpenProject(root)
	if err != nil {
		t.Fatal(err)
	}

	info, err := p.ReuseInfoOf("docs/guide.md")
	if err != nil {
		t.Fatal(err)
	}

	// Override should use TOML values, not file header.
	assertSlice(t, info.LicenseExpressions, []string{"CC-BY-SA-4.0"})
	assertSlice(t, info.CopyrightNotices, []string{"2024 Override Author"})
}

func TestProject_ReuseInfoOf_Closest(t *testing.T) {
	root := setupFakeProject(t, "toml")

	// File with no REUSE info.
	mkfile(t, root, "src/empty.go", `package src

func Empty() {}`)

	writeReuseTOML(t, root, `version = 1

[[annotations]]
path = "src/**"
SPDX-FileCopyrightText = "2024 TOML Fallback"
SPDX-License-Identifier = "MIT"
`)

	p, err := OpenProject(root)
	if err != nil {
		t.Fatal(err)
	}

	info, err := p.ReuseInfoOf("src/empty.go")
	if err != nil {
		t.Fatal(err)
	}

	// Closest should fill in from TOML since file has no info.
	assertSlice(t, info.LicenseExpressions, []string{"MIT"})
	assertSlice(t, info.CopyrightNotices, []string{"2024 TOML Fallback"})
}

func TestProject_ReuseInfoOf_ClosestPartial(t *testing.T) {
	root := setupFakeProject(t, "toml")

	// File with copyright but no license.
	mkfile(t, root, "src/partial.go", `// SPDX-FileCopyrightText: 2024 Partial Author
package src`)

	writeReuseTOML(t, root, `version = 1

[[annotations]]
path = "src/**"
SPDX-FileCopyrightText = "2024 TOML Author"
SPDX-License-Identifier = "MIT"
`)

	p, err := OpenProject(root)
	if err != nil {
		t.Fatal(err)
	}

	info, err := p.ReuseInfoOf("src/partial.go")
	if err != nil {
		t.Fatal(err)
	}

	// Copyright from file, license from TOML.
	assertSlice(t, info.CopyrightNotices, []string{"2024 Partial Author"})
	assertSlice(t, info.LicenseExpressions, []string{"MIT"})
}

func TestProject_ReuseInfoOf_Aggregate(t *testing.T) {
	root := setupFakeProject(t, "toml")

	mkfile(t, root, "src/agg.go", `// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2024 File Author
package src`)

	writeReuseTOML(t, root, `version = 1

[[annotations]]
path = "src/**"
precedence = "aggregate"
SPDX-FileCopyrightText = "2024 TOML Author"
SPDX-License-Identifier = "Apache-2.0"
`)

	p, err := OpenProject(root)
	if err != nil {
		t.Fatal(err)
	}

	info, err := p.ReuseInfoOf("src/agg.go")
	if err != nil {
		t.Fatal(err)
	}

	// Both file and TOML values should be present.
	assertSlice(t, info.LicenseExpressions, []string{"MIT", "Apache-2.0"})
	assertSlice(t, info.CopyrightNotices, []string{"2024 File Author", "2024 TOML Author"})
}

func TestProject_ReuseInfoOf_Dep5(t *testing.T) {
	root := setupFakeProject(t, "dep5")

	// File with no headers, should fall through to dep5.
	mkfile(t, root, "src/bare.go", `package src

func Bare() {}`)

	p, err := OpenProject(root)
	if err != nil {
		t.Fatal(err)
	}

	info, err := p.ReuseInfoOf("src/bare.go")
	if err != nil {
		t.Fatal(err)
	}

	assertSlice(t, info.LicenseExpressions, []string{"MIT"})
	assertSlice(t, info.CopyrightNotices, []string{"2024 Dep5 Author"})
}

func TestProject_ReuseInfoOf_SidecarOverridesAll(t *testing.T) {
	root := setupFakeProject(t, "toml")

	mkfile(t, root, "src/special.go", `// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2024 File Author
package src`)
	mkfile(t, root, "src/special.go.license", `SPDX-License-Identifier: GPL-3.0-only
SPDX-FileCopyrightText: 2024 Sidecar Author`)

	writeReuseTOML(t, root, `version = 1

[[annotations]]
path = "src/**"
precedence = "override"
SPDX-FileCopyrightText = "2024 TOML Author"
SPDX-License-Identifier = "Apache-2.0"
`)

	p, err := OpenProject(root)
	if err != nil {
		t.Fatal(err)
	}

	info, err := p.ReuseInfoOf("src/special.go")
	if err != nil {
		t.Fatal(err)
	}

	// Sidecar takes absolute precedence over everything.
	assertSlice(t, info.LicenseExpressions, []string{"GPL-3.0-only"})
	assertSlice(t, info.CopyrightNotices, []string{"2024 Sidecar Author"})
}

func TestProject_AllReuseInfo(t *testing.T) {
	root := setupFakeProject(t, "header")

	p, err := OpenProject(root)
	if err != nil {
		t.Fatal(err)
	}

	all, err := p.AllReuseInfo()
	if err != nil {
		t.Fatal(err)
	}

	if len(all) == 0 {
		t.Error("expected at least one file in AllReuseInfo")
	}

	info, ok := all["src/main.go"]
	if !ok {
		t.Fatal("expected src/main.go in results")
	}
	assertSlice(t, info.LicenseExpressions, []string{"MIT"})
}

func TestProject_ReuseExample(t *testing.T) {
	// Uses the fsfe/reuse-example submodule for real-world conformance.
	exampleDir := filepath.Join("testdata", "reuse-example")
	if _, err := os.Stat(exampleDir); os.IsNotExist(err) {
		t.Skip("testdata/reuse-example submodule not checked out")
	}

	p, err := OpenProject(exampleDir)
	if err != nil {
		t.Fatal(err)
	}

	all, err := p.AllReuseInfo()
	if err != nil {
		t.Fatal(err)
	}

	if len(all) == 0 {
		t.Fatal("expected covered files in reuse-example")
	}

	// Every covered file should have at least a license.
	for path, info := range all {
		if !info.HasLicense() {
			t.Errorf("%s: missing license", path)
		}
		if !info.HasCopyright() {
			t.Errorf("%s: missing copyright", path)
		}
	}
}

func TestProject_FakeRepository(t *testing.T) {
	fakeDir := filepath.Join("testdata", "fake_repository")
	if _, err := os.Stat(fakeDir); os.IsNotExist(err) {
		t.Skip("testdata/fake_repository not found")
	}

	p, err := OpenProject(fakeDir)
	if err != nil {
		t.Fatal(err)
	}

	if p.ReuseTOML == nil {
		t.Fatal("expected REUSE.toml")
	}

	// src/main.go has file headers and matches TOML closest.
	info, err := p.ReuseInfoOf("src/main.go")
	if err != nil {
		t.Fatal(err)
	}
	assertSlice(t, info.LicenseExpressions, []string{"MIT"})
	assertSlice(t, info.CopyrightNotices, []string{"2024 Alice"})
	assertSlice(t, info.Contributors, []string{"Bob"})

	// src/dual.go has dual licensing.
	info, err = p.ReuseInfoOf("src/dual.go")
	if err != nil {
		t.Fatal(err)
	}
	assertSlice(t, info.LicenseExpressions, []string{"MIT OR Apache-2.0"})
	assertSlice(t, info.CopyrightNotices, []string{"2024 Alice", "2023 Bob"})

	// src/noheader.go has no headers, so closest TOML fills in.
	info, err = p.ReuseInfoOf("src/noheader.go")
	if err != nil {
		t.Fatal(err)
	}
	assertSlice(t, info.LicenseExpressions, []string{"MIT"})
	assertSlice(t, info.CopyrightNotices, []string{"2024 TOML Org"})

	// src/ignore.go has REUSE-IgnoreStart/End, so only visible tags show.
	info, err = p.ReuseInfoOf("src/ignore.go")
	if err != nil {
		t.Fatal(err)
	}
	assertSlice(t, info.LicenseExpressions, []string{"MIT"})
	assertSlice(t, info.CopyrightNotices, []string{"2024 Visible"})

	// assets/logo.png has a .license sidecar, which takes priority.
	info, err = p.ReuseInfoOf("assets/logo.png")
	if err != nil {
		t.Fatal(err)
	}
	assertSlice(t, info.LicenseExpressions, []string{"CC0-1.0"})
	if info.SourceType != DotLicense {
		t.Errorf("assets/logo.png SourceType = %v, want DotLicense", info.SourceType)
	}

	// License files in LICENSES/ should be discovered.
	if len(p.LicenseFiles) < 3 {
		t.Errorf("expected at least 3 license files, got %d", len(p.LicenseFiles))
	}
}

// Test helpers.

func setupFakeProject(t *testing.T, mode string) string {
	t.Helper()
	root := t.TempDir()

	// Common files.
	mkfile(t, root, "LICENSES/MIT.txt", "MIT License text")
	mkfile(t, root, "src/main.go", `// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2024 Test Author
package main`)

	switch mode {
	case "toml":
		writeReuseTOML(t, root, `version = 1

[[annotations]]
path = "src/**"
SPDX-FileCopyrightText = "2024 TOML Author"
SPDX-License-Identifier = "MIT"
`)
	case "dep5":
		mkfile(t, root, ".reuse/dep5", `Format: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/

Files: *
Copyright: 2024 Dep5 Author
License: MIT
`)
	case "header":
		// No global licensing config; header-only.
	}

	return root
}

func writeReuseTOML(t *testing.T, root, content string) {
	t.Helper()
	mkfile(t, root, "REUSE.toml", content)
}
