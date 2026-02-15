package extract

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/git-pkgs/reuse/internal/core"
)

func TestExtractReuseInfo_CStyleComment(t *testing.T) {
	text := `/*
 * SPDX-License-Identifier: MIT
 * SPDX-FileCopyrightText: 2024 Jane Doe <jane@example.com>
 */`

	info := ExtractReuseInfo(text)
	assertSlice(t, info.LicenseExpressions, []string{"MIT"})
	assertSlice(t, info.CopyrightNotices, []string{"2024 Jane Doe <jane@example.com>"})
}

func TestExtractReuseInfo_HashComment(t *testing.T) {
	text := `# SPDX-License-Identifier: Apache-2.0
# SPDX-FileCopyrightText: 2023 Acme Corp`

	info := ExtractReuseInfo(text)
	assertSlice(t, info.LicenseExpressions, []string{"Apache-2.0"})
	assertSlice(t, info.CopyrightNotices, []string{"2023 Acme Corp"})
}

func TestExtractReuseInfo_DoubleSlashComment(t *testing.T) {
	text := `// SPDX-License-Identifier: GPL-3.0-or-later
// SPDX-FileCopyrightText: 2020 Free Software Foundation`

	info := ExtractReuseInfo(text)
	assertSlice(t, info.LicenseExpressions, []string{"GPL-3.0-or-later"})
	assertSlice(t, info.CopyrightNotices, []string{"2020 Free Software Foundation"})
}

func TestExtractReuseInfo_DashDashComment(t *testing.T) {
	text := `-- SPDX-License-Identifier: BSD-2-Clause
-- SPDX-FileCopyrightText: 2021 Some Author`

	info := ExtractReuseInfo(text)
	assertSlice(t, info.LicenseExpressions, []string{"BSD-2-Clause"})
	assertSlice(t, info.CopyrightNotices, []string{"2021 Some Author"})
}

func TestExtractReuseInfo_HTMLComment(t *testing.T) {
	text := `<!-- SPDX-License-Identifier: MIT -->
<!-- SPDX-FileCopyrightText: 2024 Web Dev -->`

	info := ExtractReuseInfo(text)
	assertSlice(t, info.LicenseExpressions, []string{"MIT"})
	assertSlice(t, info.CopyrightNotices, []string{"2024 Web Dev"})
}

func TestExtractReuseInfo_MultipleLicensesAndCopyrights(t *testing.T) {
	text := `// SPDX-License-Identifier: MIT
// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2020 Alice
// SPDX-FileCopyrightText: 2021 Bob
// SPDX-FileContributor: Charlie`

	info := ExtractReuseInfo(text)
	assertSlice(t, info.LicenseExpressions, []string{"MIT", "Apache-2.0"})
	assertSlice(t, info.CopyrightNotices, []string{"2020 Alice", "2021 Bob"})
	assertSlice(t, info.Contributors, []string{"Charlie"})
}

func TestExtractReuseInfo_ORExpression(t *testing.T) {
	text := `// SPDX-License-Identifier: MIT OR Apache-2.0`
	info := ExtractReuseInfo(text)
	assertSlice(t, info.LicenseExpressions, []string{"MIT OR Apache-2.0"})
}

func TestExtractReuseInfo_EmptyText(t *testing.T) {
	info := ExtractReuseInfo("")
	if !info.IsEmpty() {
		t.Errorf("expected empty ReuseInfo for empty text")
	}
}

func TestExtractReuseInfo_NoTags(t *testing.T) {
	text := `package main

import "fmt"

func main() {
	fmt.Println("hello")
}`
	info := ExtractReuseInfo(text)
	if !info.IsEmpty() {
		t.Errorf("expected empty ReuseInfo for text without SPDX tags")
	}
}

func TestExtractReuseInfo_SnippetCopyrightText(t *testing.T) {
	text := `// SPDX-SnippetBegin
// SPDX-License-Identifier: GPL-2.0-only
// SPDX-SnippetCopyrightText: 2019 Snippet Author
// some code here
// SPDX-SnippetEnd`

	info := ExtractReuseInfo(text)
	assertSlice(t, info.LicenseExpressions, []string{"GPL-2.0-only"})
	assertSlice(t, info.CopyrightNotices, []string{"2019 Snippet Author"})
}

func TestExtractReuseInfo_MixedFileAndSnippet(t *testing.T) {
	text := `// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2024 File Author
//
// some code
//
// SPDX-SnippetBegin
// SPDX-License-Identifier: GPL-2.0-only
// SPDX-SnippetCopyrightText: 2019 Snippet Author
// snippet code
// SPDX-SnippetEnd
//
// more code`

	info := ExtractReuseInfo(text)
	assertSlice(t, info.LicenseExpressions, []string{"MIT", "GPL-2.0-only"})
	assertSlice(t, info.CopyrightNotices, []string{"2024 File Author", "2019 Snippet Author"})
}

func TestFilterIgnoreBlocks(t *testing.T) {
	text := `before
REUSE-IgnoreStart
ignored content
REUSE-IgnoreEnd
after`

	got := FilterIgnoreBlocks(text)
	if got != "before\n\nafter" {
		t.Errorf("got %q, want %q", got, "before\n\nafter")
	}
}

func TestFilterIgnoreBlocks_Unclosed(t *testing.T) {
	text := `before
REUSE-IgnoreStart
ignored to end`

	got := FilterIgnoreBlocks(text)
	if got != "before\n" {
		t.Errorf("got %q, want %q", got, "before\n")
	}
}

func TestFilterIgnoreBlocks_Multiple(t *testing.T) {
	text := `a
REUSE-IgnoreStart
b
REUSE-IgnoreEnd
c
REUSE-IgnoreStart
d
REUSE-IgnoreEnd
e`

	got := FilterIgnoreBlocks(text)
	if got != "a\n\nc\n\ne" {
		t.Errorf("got %q, want %q", got, "a\n\nc\n\ne")
	}
}

func TestFilterIgnoreBlocks_None(t *testing.T) {
	text := "no ignore blocks here"
	got := FilterIgnoreBlocks(text)
	if got != text {
		t.Errorf("got %q, want %q", got, text)
	}
}

func TestExtractReuseInfo_IgnoreBlockHidesTag(t *testing.T) {
	text := `// SPDX-License-Identifier: MIT
REUSE-IgnoreStart
// SPDX-License-Identifier: GPL-3.0-only
REUSE-IgnoreEnd
// SPDX-FileCopyrightText: 2024 Author`

	info := ExtractReuseInfo(text)
	assertSlice(t, info.LicenseExpressions, []string{"MIT"})
	assertSlice(t, info.CopyrightNotices, []string{"2024 Author"})
}

func TestExtractReuseInfo_WindowsLineEndings(t *testing.T) {
	text := "// SPDX-License-Identifier: MIT\r\n// SPDX-FileCopyrightText: 2024 Author\r\n"
	info := ExtractReuseInfo(text)
	assertSlice(t, info.LicenseExpressions, []string{"MIT"})
	assertSlice(t, info.CopyrightNotices, []string{"2024 Author"})
}

func TestExtractReuseInfo_TrailingCommentMarker(t *testing.T) {
	text := `/* SPDX-License-Identifier: MIT */
/* SPDX-FileCopyrightText: 2024 Author */`

	info := ExtractReuseInfo(text)
	assertSlice(t, info.LicenseExpressions, []string{"MIT"})
	assertSlice(t, info.CopyrightNotices, []string{"2024 Author"})
}

func TestExtractFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.go")
	content := `// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2024 Test
package main`

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := ExtractFromFile(path)
	if err != nil {
		t.Fatal(err)
	}

	assertSlice(t, info.LicenseExpressions, []string{"MIT"})
	assertSlice(t, info.CopyrightNotices, []string{"2024 Test"})
	if info.SourcePath != path {
		t.Errorf("SourcePath = %q, want %q", info.SourcePath, path)
	}
	if info.SourceType != core.FileHeader {
		t.Errorf("SourceType = %v, want FileHeader", info.SourceType)
	}
}

func TestExtractFromFile_Binary(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "image.png")
	// Write data with null bytes to simulate binary.
	data := []byte{0x89, 0x50, 0x4E, 0x47, 0x00, 0x0D, 0x0A, 0x1A}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	info, err := ExtractFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !info.IsEmpty() {
		t.Errorf("expected empty ReuseInfo for binary file")
	}
}

func TestExtractFromFile_NotFound(t *testing.T) {
	_, err := ExtractFromFile("/nonexistent/file.go")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestExtractReuseInfo_MalformedTag(t *testing.T) {
	// Missing colon after tag name should not match.
	text := `// SPDX-License-Identifier MIT
// SPDX-FileCopyrightText 2024 Author`

	info := ExtractReuseInfo(text)
	if !info.IsEmpty() {
		t.Errorf("expected empty ReuseInfo for malformed tags, got licenses=%v copyrights=%v",
			info.LicenseExpressions, info.CopyrightNotices)
	}
}

func TestExtractReuseInfo_LeadingWhitespace(t *testing.T) {
	text := `   // SPDX-License-Identifier: MIT
   // SPDX-FileCopyrightText: 2024 Author`

	info := ExtractReuseInfo(text)
	assertSlice(t, info.LicenseExpressions, []string{"MIT"})
	assertSlice(t, info.CopyrightNotices, []string{"2024 Author"})
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
