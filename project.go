package reuse

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/git-pkgs/reuse/dep5"
	"github.com/git-pkgs/reuse/extract"
	rtoml "github.com/git-pkgs/reuse/toml"
)

// Project represents a REUSE-compliant project directory.
type Project struct {
	Root         string
	ReuseTOML    *rtoml.ReuseTOML
	Dep5         *dep5.Dep5
	LicenseFiles []string // relative paths in LICENSES/
}

// OpenProject discovers and parses REUSE metadata in the given project root.
// It looks for REUSE.toml, .reuse/dep5, and LICENSES/ directory.
func OpenProject(root string) (*Project, error) {
	p := &Project{Root: root}

	// Try REUSE.toml.
	tomlPath := filepath.Join(root, "REUSE.toml")
	if _, err := os.Stat(tomlPath); err == nil {
		rt, err := rtoml.ParseReuseTOMLFile(tomlPath)
		if err != nil {
			return nil, err
		}
		p.ReuseTOML = rt
	}

	// Try .reuse/dep5 (only if no REUSE.toml, they are mutually exclusive).
	if p.ReuseTOML == nil {
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

// ReuseInfoOf resolves all REUSE information sources for the given path
// (relative to project root) and returns the combined result. The precedence
// order follows the spec:
//
//  1. .license sidecar file (if present, used exclusively)
//  2. REUSE.toml override annotations
//  3. File header extraction
//  4. REUSE.toml closest/aggregate annotations
//  5. dep5
func (p *Project) ReuseInfoOf(path string) (ReuseInfo, error) {
	// 1. Check for .license sidecar.
	sidecarPath := filepath.Join(p.Root, path+".license")
	if fi, err := os.Stat(sidecarPath); err == nil && fi.Mode().IsRegular() {
		info, err := extract.ExtractFromFile(sidecarPath)
		if err != nil {
			return ReuseInfo{}, err
		}
		info.SourceType = DotLicense
		info.SourcePath = path + ".license"
		return info, nil
	}

	// 2. Check REUSE.toml for override.
	if p.ReuseTOML != nil {
		tomlInfo, prec, ok := p.ReuseTOML.ReuseInfoOf(path)
		if ok && prec == Override {
			return tomlInfo, nil
		}
	}

	// 3. Extract from file header.
	filePath := filepath.Join(p.Root, path)
	fileInfo, err := extract.ExtractFromFile(filePath)
	if err != nil {
		// If the file can't be read, still try other sources.
		if !os.IsNotExist(err) {
			return ReuseInfo{}, err
		}
		fileInfo = ReuseInfo{}
	}
	fileInfo.SourcePath = path
	fileInfo.SourceType = FileHeader

	// 4. Apply REUSE.toml closest/aggregate annotations.
	if p.ReuseTOML != nil {
		tomlInfo, prec, ok := p.ReuseTOML.ReuseInfoOf(path)
		if ok {
			switch prec {
			case Aggregate:
				fileInfo = mergeReuseInfo(fileInfo, tomlInfo)
			case Closest:
				fileInfo = applyClosest(fileInfo, tomlInfo)
			}
		}
	}

	// 5. Check dep5 (aggregate precedence).
	if p.Dep5 != nil {
		dep5Info, ok := p.Dep5.ReuseInfoOf(path)
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
	base.LicenseExpressions = appendUnique(base.LicenseExpressions, other.LicenseExpressions...)
	base.CopyrightNotices = appendUnique(base.CopyrightNotices, other.CopyrightNotices...)
	base.Contributors = appendUnique(base.Contributors, other.Contributors...)
	return base
}

// applyClosest fills in missing license or copyright from the REUSE.toml
// annotation, but only if the file is missing that particular piece.
func applyClosest(fileInfo, tomlInfo ReuseInfo) ReuseInfo {
	if !fileInfo.HasLicense() && !fileInfo.HasCopyright() {
		// File has no REUSE info at all; use the toml info entirely.
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
