package reuse

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	ignoreDirPatterns = []*regexp.Regexp{
		regexp.MustCompile(`^\.git$`),
		regexp.MustCompile(`^\.hg$`),
		regexp.MustCompile(`^\.sl$`),
		regexp.MustCompile(`^LICENSES$`),
		regexp.MustCompile(`^\.reuse$`),
	}

	ignoreFilePatterns = []*regexp.Regexp{
		regexp.MustCompile(`^LICEN[CS]E([-.].*)?$`),
		regexp.MustCompile(`^COPYING([-.].*)?$`),
		regexp.MustCompile(`^\.git$`),
		regexp.MustCompile(`^\.hgtags$`),
		regexp.MustCompile(`.*\.license$`),
		regexp.MustCompile(`^REUSE\.toml$`),
		regexp.MustCompile(`.*\.spdx$`),
		regexp.MustCompile(`.*\.spdx\.(rdf|json|xml|ya?ml)$`),
	}
)

// IsIgnoredDir returns true if the directory name should be skipped when
// walking a project for covered files.
func IsIgnoredDir(name string) bool {
	for _, re := range ignoreDirPatterns {
		if re.MatchString(name) {
			return true
		}
	}
	return false
}

// IsIgnoredFile returns true if the file name should be excluded from covered
// file analysis per the REUSE spec.
func IsIgnoredFile(name string) bool {
	for _, re := range ignoreFilePatterns {
		if re.MatchString(name) {
			return true
		}
	}
	return false
}

// IsCoveredFile checks whether a path (relative to the project root) represents
// a file that needs licensing information according to the REUSE spec.
func IsCoveredFile(path string) bool {
	name := filepath.Base(path)

	if IsIgnoredFile(name) {
		return false
	}

	// Check if any directory component is ignored.
	dir := filepath.Dir(path)
	if dir != "." {
		for _, part := range strings.Split(dir, string(filepath.Separator)) {
			if IsIgnoredDir(part) {
				return false
			}
		}
	}

	return true
}

// CoveredFiles walks a project directory and returns the relative paths of all
// files that need licensing information. It skips ignored directories, symlinks,
// zero-byte files, and files matching ignore patterns.
func CoveredFiles(root string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		name := d.Name()

		if d.IsDir() {
			if path == root {
				return nil
			}
			if IsIgnoredDir(name) {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip symlinks.
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		if IsIgnoredFile(name) {
			return nil
		}

		// Skip zero-byte files.
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Size() == 0 {
			return nil
		}

		// Skip symlinks detected via Lstat (WalkDir may resolve them).
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}

		files = append(files, rel)
		return nil
	})

	return files, err
}
