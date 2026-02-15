package dep5

import (
	"fmt"
	"os"
	"strings"

	"github.com/git-pkgs/reuse/internal/core"
)

// Dep5 represents a parsed .reuse/dep5 file in Debian copyright format 1.0.
type Dep5 struct {
	Header Dep5Header
	Files  []Dep5FilesParagraph
}

// Dep5Header is the first paragraph of a dep5 file.
type Dep5Header struct {
	Format string
}

// Dep5FilesParagraph is a Files paragraph specifying licensing for a set of paths.
type Dep5FilesParagraph struct {
	Patterns  []string
	Copyright string
	License   string
}

// ParseDep5 parses a .reuse/dep5 file from its content string.
func ParseDep5(content string) (*Dep5, error) {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")

	paragraphs := splitParagraphs(content)
	if len(paragraphs) == 0 {
		return nil, fmt.Errorf("dep5: empty file")
	}

	d := &Dep5{}

	// Parse header paragraph.
	headerFields := parseDep5Fields(paragraphs[0])
	d.Header.Format = headerFields["Format"]

	// Parse file paragraphs.
	for i, para := range paragraphs[1:] {
		fields := parseDep5Fields(para)

		filesStr := fields["Files"]
		if filesStr == "" {
			return nil, fmt.Errorf("dep5: paragraph %d missing Files field", i+1)
		}

		copyright := fields["Copyright"]
		if copyright == "" {
			return nil, fmt.Errorf("dep5: paragraph %d missing Copyright field", i+1)
		}

		license := fields["License"]
		if license == "" {
			return nil, fmt.Errorf("dep5: paragraph %d missing License field", i+1)
		}

		patterns := strings.Fields(filesStr)

		d.Files = append(d.Files, Dep5FilesParagraph{
			Patterns:  patterns,
			Copyright: copyright,
			License:   license,
		})
	}

	return d, nil
}

// ParseDep5File reads and parses a .reuse/dep5 file from disk.
func ParseDep5File(path string) (*Dep5, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseDep5(string(data))
}

// ReuseInfoOf finds the last matching Files paragraph for the given path and
// returns its licensing information. dep5 uses aggregate precedence.
func (d *Dep5) ReuseInfoOf(path string) (core.ReuseInfo, bool) {
	var match *Dep5FilesParagraph

	for i := range d.Files {
		for _, pattern := range d.Files[i].Patterns {
			if dep5Match(pattern, path) {
				match = &d.Files[i]
				break
			}
		}
	}

	if match == nil {
		return core.ReuseInfo{}, false
	}

	return core.ReuseInfo{
		LicenseExpressions: []string{match.License},
		CopyrightNotices:   splitDep5Copyright(match.Copyright),
		SourceType:         core.Dep5Source,
	}, true
}

// splitParagraphs splits dep5 content into paragraphs separated by blank lines.
func splitParagraphs(text string) []string {
	var paragraphs []string
	var current strings.Builder

	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == "" {
			if current.Len() > 0 {
				paragraphs = append(paragraphs, current.String())
				current.Reset()
			}
			continue
		}
		if current.Len() > 0 {
			current.WriteByte('\n')
		}
		current.WriteString(line)
	}

	if current.Len() > 0 {
		paragraphs = append(paragraphs, current.String())
	}

	return paragraphs
}

// parseDep5Fields parses a paragraph into field name/value pairs, handling
// continuation lines (lines starting with whitespace or ".").
func parseDep5Fields(paragraph string) map[string]string {
	fields := make(map[string]string)
	var currentKey string
	var currentValue strings.Builder

	for _, line := range strings.Split(paragraph, "\n") {
		if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
			// Continuation line.
			if currentKey != "" {
				trimmed := strings.TrimSpace(line)
				if trimmed == "." {
					currentValue.WriteByte('\n')
				} else {
					currentValue.WriteByte('\n')
					currentValue.WriteString(trimmed)
				}
			}
			continue
		}

		// Save previous field.
		if currentKey != "" {
			fields[currentKey] = strings.TrimSpace(currentValue.String())
		}

		// Parse new field.
		if idx := strings.IndexByte(line, ':'); idx >= 0 {
			currentKey = line[:idx]
			currentValue.Reset()
			currentValue.WriteString(strings.TrimSpace(line[idx+1:]))
		}
	}

	if currentKey != "" {
		fields[currentKey] = strings.TrimSpace(currentValue.String())
	}

	return fields
}

// dep5Match checks if a path matches a dep5 glob pattern.
// dep5 uses fnmatch-style patterns where * matches everything including /.
func dep5Match(pattern, path string) bool {
	// dep5 uses fnmatch with FNM_PATHNAME not set, so * matches /.
	return dep5MatchRecursive(pattern, path)
}

func dep5MatchRecursive(pattern, name string) bool {
	for len(pattern) > 0 {
		switch pattern[0] {
		case '*':
			pattern = pattern[1:]
			// Try matching the rest from every position.
			for i := 0; i <= len(name); i++ {
				if dep5MatchRecursive(pattern, name[i:]) {
					return true
				}
			}
			return false
		case '?':
			if len(name) == 0 {
				return false
			}
			pattern = pattern[1:]
			name = name[1:]
		default:
			if len(name) == 0 || pattern[0] != name[0] {
				return false
			}
			pattern = pattern[1:]
			name = name[1:]
		}
	}
	return len(name) == 0
}

// splitDep5Copyright splits a multi-line copyright field into individual notices.
func splitDep5Copyright(copyright string) []string {
	var notices []string
	for _, line := range strings.Split(copyright, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			notices = append(notices, line)
		}
	}
	return notices
}
