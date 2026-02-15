package extract

import (
	"os"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/git-pkgs/reuse/internal/core"
)

var (
	// Trailing comment-end markers stripped from matched values.
	endPattern = `[\s]*(?:\*/|-->|"|'>|\]\s*::|"\))?[\s]*$`

	licensePattern = regexp.MustCompile(
		`(?m)^.*?SPDX-License-Identifier:\s*(?P<value>.*?)` + endPattern,
	)

	copyrightPattern = regexp.MustCompile(
		`(?m)^.*?SPDX-(?:File|Snippet)CopyrightText:\s*(?P<value>.*?)` + endPattern,
	)

	contributorPattern = regexp.MustCompile(
		`(?m)^.*?SPDX-FileContributor:\s*(?P<value>.*?)` + endPattern,
	)

	snippetBeginPattern = regexp.MustCompile(`(?m)^.*?SPDX-SnippetBegin`)
	snippetEndPattern   = regexp.MustCompile(`(?m)^.*?SPDX-SnippetEnd`)
)

// ExtractReuseInfo extracts SPDX license, copyright, and contributor
// information from the given text. It handles REUSE-IgnoreStart/End blocks
// and SPDX-SnippetBegin/End regions.
func ExtractReuseInfo(text string) core.ReuseInfo {
	// Normalize line endings.
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	// Strip ignore blocks.
	text = FilterIgnoreBlocks(text)

	// Strip snippet regions but keep the snippet tags themselves for extraction.
	// The spec says snippet info applies only to the snippet, but for file-level
	// extraction we collect everything outside snippets plus snippet declarations.
	fileText, snippetTexts := splitSnippets(text)

	var info core.ReuseInfo
	extractFromText(fileText, &info)
	for _, st := range snippetTexts {
		extractFromText(st, &info)
	}

	return info
}

func extractFromText(text string, info *core.ReuseInfo) {
	for _, m := range licensePattern.FindAllStringSubmatch(text, -1) {
		val := strings.TrimSpace(m[1])
		if val != "" {
			info.LicenseExpressions = append(info.LicenseExpressions, val)
		}
	}
	for _, m := range copyrightPattern.FindAllStringSubmatch(text, -1) {
		val := strings.TrimSpace(m[1])
		if val != "" {
			info.CopyrightNotices = append(info.CopyrightNotices, val)
		}
	}
	for _, m := range contributorPattern.FindAllStringSubmatch(text, -1) {
		val := strings.TrimSpace(m[1])
		if val != "" {
			info.Contributors = append(info.Contributors, val)
		}
	}
}

// splitSnippets separates text into the parts outside SPDX-SnippetBegin/End
// regions and the snippet regions themselves. If no snippets are found, the
// entire text is returned as the file portion.
func splitSnippets(text string) (string, []string) {
	beginLocs := snippetBeginPattern.FindAllStringIndex(text, -1)
	if len(beginLocs) == 0 {
		return text, nil
	}

	endLocs := snippetEndPattern.FindAllStringIndex(text, -1)

	var fileParts []string
	var snippets []string
	pos := 0
	endIdx := 0

	for _, begin := range beginLocs {
		if begin[0] > pos {
			fileParts = append(fileParts, text[pos:begin[0]])
		}

		// Find the matching end after this begin.
		snippetEnd := len(text)
		for endIdx < len(endLocs) {
			if endLocs[endIdx][1] > begin[1] {
				snippetEnd = endLocs[endIdx][1]
				endIdx++
				break
			}
			endIdx++
		}

		snippets = append(snippets, text[begin[0]:snippetEnd])
		pos = snippetEnd
	}

	if pos < len(text) {
		fileParts = append(fileParts, text[pos:])
	}

	return strings.Join(fileParts, "\n"), snippets
}

// FilterIgnoreBlocks removes text between REUSE-IgnoreStart and
// REUSE-IgnoreEnd markers. An unclosed block removes everything from the
// start marker to the end of text.
func FilterIgnoreBlocks(text string) string {
	const (
		startMarker = "REUSE-IgnoreStart"
		endMarker   = "REUSE-IgnoreEnd"
	)

	if !strings.Contains(text, startMarker) {
		return text
	}

	var b strings.Builder
	for {
		startIdx := strings.Index(text, startMarker)
		if startIdx == -1 {
			b.WriteString(text)
			break
		}

		b.WriteString(text[:startIdx])

		rest := text[startIdx+len(startMarker):]
		endIdx := strings.Index(rest, endMarker)
		if endIdx == -1 {
			// Unclosed block: discard the rest.
			break
		}

		text = rest[endIdx+len(endMarker):]
	}

	return b.String()
}

// ExtractFromFile reads a file and extracts REUSE information from its contents.
// Binary files (detected by invalid UTF-8 or null bytes) return an empty ReuseInfo.
func ExtractFromFile(path string) (core.ReuseInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return core.ReuseInfo{}, err
	}

	if isBinary(data) {
		return core.ReuseInfo{}, nil
	}

	info := ExtractReuseInfo(string(data))
	info.SourcePath = path
	info.SourceType = core.FileHeader
	return info, nil
}

// isBinary returns true if data looks like a binary file. It checks for null
// bytes and invalid UTF-8 in the first 8KB.
func isBinary(data []byte) bool {
	check := data
	if len(check) > 8192 {
		check = check[:8192]
	}

	for _, b := range check {
		if b == 0 {
			return true
		}
	}

	return !utf8.Valid(check)
}
