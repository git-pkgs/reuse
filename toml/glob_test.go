package toml

import "testing"

func TestGlobMatch(t *testing.T) {
	tests := []struct {
		pattern string
		path    string
		want    bool
	}{
		// Single star matches within a directory
		{"*.go", "foo.go", true},
		{"*.go", "bar.go", true},
		{"*.go", "dir/foo.go", false},
		{"*.go", ".hidden.go", true},

		// Star in a directory prefix
		{"src/*.py", "src/foo.py", true},
		{"src/*.py", "src/other/foo.py", false},

		// Double star matches across directories
		{"**/*.go", "foo.go", true},
		{"**/*.go", "dir/foo.go", true},
		{"**/*.go", "dir/sub/foo.go", true},
		{"**/*.go", ".hidden/foo.go", true},

		// Double star as full pattern
		{"**", "anything", true},
		{"**", "dir/sub/file.txt", true},

		// Double star prefix
		{"src/**", "src/foo.go", true},
		{"src/**", "src/sub/foo.go", true},
		{"src/**", "other/foo.go", false},

		// Literal matching
		{"foo.py", "foo.py", true},
		{"foo.py", "src/foo.py", false},
		{"src/foo.py", "src/foo.py", true},
		{"src/foo.py", "foo.py", false},

		// Single star matches everything except /
		{"*", "foo.py", true},
		{"*", ".gitignore", true},
		{"*", "src/foo.py", false},

		// Escaped asterisk
		{`\*.py`, "*.py", true},
		{`\*.py`, "foo.py", false},

		// Escaped backslash followed by star
		{`\\*.py`, `\foo.py`, true},
		{`\\*.py`, `foo.py`, false},

		// Star in the middle
		{"foo*bar", "foobar", true},
		{"foo*bar", "foo2bar", true},
		{"foo*bar", "foo/bar", false},

		// Question mark
		{"foo?.go", "foo1.go", true},
		{"foo?.go", "fooab.go", false},
		{"foo?.go", "foo/.go", false},

		// Empty pattern and path
		{"", "", true},
		{"*", "", true},
		{"?", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.path, func(t *testing.T) {
			got := GlobMatch(tt.pattern, tt.path)
			if got != tt.want {
				t.Errorf("GlobMatch(%q, %q) = %v, want %v", tt.pattern, tt.path, got, tt.want)
			}
		})
	}
}
