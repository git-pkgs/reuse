package extract_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/git-pkgs/reuse/extract"
)

func TestExtractReuseInfoEmptyTags(t *testing.T) {
	for _, ending := range []struct{ name, text string }{
		{"LF", "\n"}, {"CRLF", "\r\n"}, {"CR", "\r"},
	} {
		for _, tag := range []string{
			"SPDX-License-Identifier", "SPDX-FileCopyrightText",
			"SPDX-SnippetCopyrightText", "SPDX-FileContributor",
		} {
			t.Run(ending.name+"/"+tag, func(t *testing.T) {
				text := strings.Join([]string{
					tag + ": \t", "", tag + ": valid", tag + ":", "ordinary text",
				}, ending.text)
				info := extract.ExtractReuseInfo(text)
				values := slices.Concat(info.LicenseExpressions, info.CopyrightNotices, info.Contributors)
				if !slices.Equal(values, []string{"valid"}) {
					t.Fatalf("values = %q, want [valid]", values)
				}
			})
		}
	}
}
