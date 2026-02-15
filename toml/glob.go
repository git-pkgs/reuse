package toml

import (
	"regexp"
	"strings"
)

// GlobMatch tests whether path matches a REUSE.toml glob pattern.
//
// Pattern rules per the spec:
//   - * matches everything except /
//   - ** matches everything including /
//   - \* is a literal asterisk
//   - \\ is a literal backslash
//   - Forward slashes only (no backslash path separators)
func GlobMatch(pattern, path string) bool {
	re := globToRegexp(pattern)
	return re.MatchString(path)
}

func globToRegexp(pattern string) *regexp.Regexp {
	var b strings.Builder
	b.WriteString("^")

	i := 0
	for i < len(pattern) {
		ch := pattern[i]
		switch {
		case ch == '\\' && i+1 < len(pattern):
			next := pattern[i+1]
			if next == '*' {
				b.WriteString(regexp.QuoteMeta("*"))
			} else if next == '\\' {
				b.WriteString(regexp.QuoteMeta("\\"))
			} else {
				b.WriteString(regexp.QuoteMeta(string(next)))
			}
			i += 2
		case ch == '*':
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				// Consume trailing slash so **/ can match empty (no directory).
				if i+2 < len(pattern) && pattern[i+2] == '/' {
					b.WriteString("(.*/)?")
					i += 3
				} else {
					b.WriteString(".*")
					i += 2
				}
			} else {
				b.WriteString("[^/]*")
				i++
			}
		case ch == '?':
			b.WriteString("[^/]")
			i++
		default:
			b.WriteString(regexp.QuoteMeta(string(ch)))
			i++
		}
	}

	b.WriteString("$")
	return regexp.MustCompile(b.String())
}
