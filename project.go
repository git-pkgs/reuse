package reuse

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"

	"github.com/git-pkgs/reuse/dep5"
	"github.com/git-pkgs/reuse/extract"
	rtoml "github.com/git-pkgs/reuse/toml"
)

// Project represents a REUSE-compliant project directory.
type Project struct {
	Root         string
	ReuseTOMLs   map[string]*rtoml.ReuseTOML // keyed by relative directory; "." is the root
	Dep5         *dep5.Dep5
	LicenseFiles []string // relative paths in LICENSES/
}

// OpenProject discovers and parses REUSE metadata in the given project root.
// It loads REUSE.toml files throughout the covered directories, or .reuse/dep5
// when no TOML files are present, and lists LICENSES/.
func OpenProject(root string) (*Project, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("project root is not a directory: %s", root)
	}
	tomls, err := readReuseTOMLs(root)
	if err != nil {
		return nil, err
	}
	p := &Project{Root: root, ReuseTOMLs: tomls}

	// Try .reuse/dep5 (only if no REUSE.toml, they are mutually exclusive).
	if len(p.ReuseTOMLs) == 0 {
		dep5Path := filepath.Join(root, ".reuse", "dep5")
		if _, err := os.Stat(dep5Path); err == nil {
			d, err := dep5.ParseDep5File(dep5Path)
			if err != nil {
				return nil, err
			}
			p.Dep5 = d
		}
	}

	// Scan LICENSES/ directory.
	licensesDir := filepath.Join(root, "LICENSES")
	entries, err := os.ReadDir(licensesDir)
	if err == nil {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			if strings.HasSuffix(name, ".license") {
				continue
			}
			p.LicenseFiles = append(p.LicenseFiles, filepath.Join("LICENSES", name))
		}
	}

	return p, nil
}

func readReuseTOMLs(root string) (map[string]*rtoml.ReuseTOML, error) {
	files := make(map[string]*rtoml.ReuseTOML)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != root && IsIgnoredDir(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Name() != "REUSE.toml" || !entry.Type().IsRegular() {
			return nil
		}
		parsed, err := rtoml.ParseReuseTOMLFile(path)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		parsed.Source = filepath.ToSlash(relative)
		files[filepath.ToSlash(filepath.Dir(relative))] = parsed
		return nil
	})
	return files, err
}

// ReuseInfoOf resolves all REUSE information sources for the given path
// (relative to project root) and returns the combined result. A sidecar replaces
// the file header. The outermost TOML override suppresses closer sources;
// otherwise closest annotations fill missing fields and aggregate adds values.
func (p *Project) ReuseInfoOf(path string) (ReuseInfo, error) {
	if !filepath.IsLocal(path) {
		return ReuseInfo{}, fmt.Errorf("path must be relative to the project: %s", path)
	}
	path = filepath.Clean(path)
	closest, aggregate, override := p.annotationsFor(path)
	if override != nil {
		return mergeReuseInfo(*override, aggregate), nil
	}
	fileInfo, err := p.fileInfo(path)
	if err != nil {
		return ReuseInfo{}, err
	}
	if !closest.IsEmpty() {
		fileInfo = applyClosest(fileInfo, closest)
	}
	fileInfo = mergeReuseInfo(fileInfo, aggregate)

	// DEP5 uses aggregate precedence.
	if p.Dep5 != nil {
		dep5Info, ok := p.Dep5.ReuseInfoOf(filepath.ToSlash(path))
		if ok {
			dep5Info.SourcePath = filepath.Join(".reuse", "dep5")
			if fileInfo.IsEmpty() {
				return dep5Info, nil
			}
			fileInfo = mergeReuseInfo(fileInfo, dep5Info)
		}
	}

	return fileInfo, nil
}

func (p *Project) annotationsFor(path string) (closest, aggregate ReuseInfo, override *ReuseInfo) {
	var directories []string
	for dir := filepath.Dir(path); ; dir = filepath.Dir(dir) {
		directories = append(directories, dir)
		if dir == "." {
			break
		}
	}
	slices.Reverse(directories)
	for _, dir := range directories {
		config := p.ReuseTOMLs[filepath.ToSlash(dir)]
		if config == nil {
			continue
		}
		relative := filepath.ToSlash(path)
		if dir != "." {
			relative = strings.TrimPrefix(relative, filepath.ToSlash(dir)+"/")
		}
		info, precedence, ok := config.ReuseInfoOf(relative)
		if !ok {
			continue
		}
		switch precedence {
		case Closest:
			closest = applyClosest(info, closest)
		case Aggregate:
			aggregate = mergeReuseInfo(aggregate, info)
		case Override:
			return ReuseInfo{}, aggregate, &info
		}
	}
	return closest, aggregate, nil
}

func (p *Project) fileInfo(path string) (ReuseInfo, error) {
	source := path
	sourceType := FileHeader
	if info, err := os.Stat(filepath.Join(p.Root, path+".license")); err == nil && info.Mode().IsRegular() {
		source += ".license"
		sourceType = DotLicense
	} else if err != nil && !os.IsNotExist(err) && !errors.Is(err, syscall.ENAMETOOLONG) {
		return ReuseInfo{}, err
	}
	info, err := extract.ExtractFromFile(filepath.Join(p.Root, source))
	if err != nil && (sourceType == DotLicense || !os.IsNotExist(err)) {
		return ReuseInfo{}, err
	}
	info.SourcePath = filepath.ToSlash(source)
	info.SourceType = sourceType
	return info, nil
}

// AllReuseInfo walks all covered files in the project and returns licensing
// information for each one, keyed by relative path.
func (p *Project) AllReuseInfo() (map[string]ReuseInfo, error) {
	files, err := CoveredFiles(p.Root)
	if err != nil {
		return nil, err
	}

	result := make(map[string]ReuseInfo, len(files))
	for _, f := range files {
		info, err := p.ReuseInfoOf(f)
		if err != nil {
			return nil, err
		}
		result[f] = info
	}

	return result, nil
}

// mergeReuseInfo combines two ReuseInfo values, appending licenses, copyrights,
// and contributors from other into base.
func mergeReuseInfo(base, other ReuseInfo) ReuseInfo {
	if base.IsEmpty() && len(base.Contributors) == 0 && !other.IsEmpty() {
		base.SourcePath, base.SourceType = other.SourcePath, other.SourceType
	}
	base.LicenseExpressions = appendUnique(base.LicenseExpressions, other.LicenseExpressions...)
	base.CopyrightNotices = appendUnique(base.CopyrightNotices, other.CopyrightNotices...)
	base.Contributors = appendUnique(base.Contributors, other.Contributors...)
	return base
}

// applyClosest fills in missing license or copyright from the REUSE.toml
// annotation, but only if the file is missing that particular piece.
func applyClosest(fileInfo, tomlInfo ReuseInfo) ReuseInfo {
	if !fileInfo.HasLicense() && !fileInfo.HasCopyright() {
		tomlInfo.Contributors = appendUnique(fileInfo.Contributors, tomlInfo.Contributors...)
		return tomlInfo
	}
	if !fileInfo.HasLicense() {
		fileInfo.LicenseExpressions = tomlInfo.LicenseExpressions
	}
	if !fileInfo.HasCopyright() {
		fileInfo.CopyrightNotices = tomlInfo.CopyrightNotices
	}
	return fileInfo
}

func appendUnique(base []string, items ...string) []string {
	if len(items) == 0 {
		return base
	}
	seen := make(map[string]bool, len(base))
	for _, s := range base {
		seen[s] = true
	}
	for _, s := range items {
		if !seen[s] {
			base = append(base, s)
			seen[s] = true
		}
	}
	return base
}
