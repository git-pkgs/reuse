package reuse

import (
	"io/fs"
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
// files that need licensing information. It skips ignored directories,
// non-regular files, zero-byte files, and files matching ignore patterns.
func CoveredFiles(root string) ([]string, error) {
	files, err := CoveredFilesFS(os.DirFS(root))
	for i := range files {
		files[i] = filepath.FromSlash(files[i])
	}
	return files, err
}

// CoveredFilesFS returns slash-separated paths of covered files in fsys.
// It skips ignored directories, non-regular files, empty files, and ignored names.
func CoveredFilesFS(fsys fs.FS) ([]string, error) {
	var files []string

	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		name := d.Name()

		if d.IsDir() {
			if path == "." {
				return nil
			}
			if IsIgnoredDir(name) {
				return fs.SkipDir
			}
			return nil
		}

		if !d.Type().IsRegular() {
			return nil
		}

		if IsIgnoredFile(name) {
			return nil
		}

		// Skip zero-byte files.
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Size() == 0 || !info.Mode().IsRegular() {
			return nil
		}

		files = append(files, path)
		return nil
	})

	return files, err
}
