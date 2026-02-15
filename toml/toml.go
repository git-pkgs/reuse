package toml

import (
	"fmt"
	"os"

	btoml "github.com/BurntSushi/toml"
	"github.com/git-pkgs/reuse/internal/core"
)

// ReuseTOML represents a parsed REUSE.toml file.
type ReuseTOML struct {
	Version     int
	Annotations []Annotation
	Source      string // file path this was loaded from
}

// Annotation is a single [[annotations]] entry in a REUSE.toml file.
type Annotation struct {
	Paths      []string
	Precedence core.PrecedenceType
	Copyrights []string
	Licenses   []string
}

// tomlFile is the raw TOML structure before conversion to our types.
type tomlFile struct {
	Version     int              `toml:"version"`
	Annotations []tomlAnnotation `toml:"annotations"`
}

type tomlAnnotation struct {
	Path       stringOrSlice `toml:"path"`
	Precedence string        `toml:"precedence"`
	Copyright  stringOrSlice `toml:"SPDX-FileCopyrightText"`
	License    stringOrSlice `toml:"SPDX-License-Identifier"`
}

// stringOrSlice handles TOML fields that can be either a string or an array of strings.
type stringOrSlice []string

func (s *stringOrSlice) UnmarshalTOML(data any) error {
	switch v := data.(type) {
	case string:
		*s = []string{v}
	case []any:
		for _, item := range v {
			str, ok := item.(string)
			if !ok {
				return fmt.Errorf("expected string in array, got %T", item)
			}
			*s = append(*s, str)
		}
	default:
		return fmt.Errorf("expected string or array, got %T", data)
	}
	return nil
}

// ParseReuseTOML parses a REUSE.toml file from its content string.
func ParseReuseTOML(content string) (*ReuseTOML, error) {
	var raw tomlFile
	if err := btoml.Unmarshal([]byte(content), &raw); err != nil {
		return nil, fmt.Errorf("reuse.toml: %w", err)
	}

	if raw.Version != 1 {
		return nil, fmt.Errorf("reuse.toml: unsupported version %d (expected 1)", raw.Version)
	}

	result := &ReuseTOML{
		Version: raw.Version,
	}

	for i, ann := range raw.Annotations {
		if len(ann.Path) == 0 {
			return nil, fmt.Errorf("reuse.toml: annotation %d has no paths", i)
		}

		prec, err := parsePrecedence(ann.Precedence)
		if err != nil {
			return nil, fmt.Errorf("reuse.toml: annotation %d: %w", i, err)
		}

		result.Annotations = append(result.Annotations, Annotation{
			Paths:      ann.Path,
			Precedence: prec,
			Copyrights: ann.Copyright,
			Licenses:   ann.License,
		})
	}

	return result, nil
}

// ParseReuseTOMLFile reads and parses a REUSE.toml file from disk.
func ParseReuseTOMLFile(path string) (*ReuseTOML, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	result, err := ParseReuseTOML(string(data))
	if err != nil {
		return nil, err
	}
	result.Source = path
	return result, nil
}

// Matches returns true if the given path matches any of the annotation's glob patterns.
func (a *Annotation) Matches(path string) bool {
	for _, pattern := range a.Paths {
		if GlobMatch(pattern, path) {
			return true
		}
	}
	return false
}

// ReuseInfoOf finds the last matching annotation for the given path and returns
// its licensing information along with the precedence type.
func (t *ReuseTOML) ReuseInfoOf(path string) (core.ReuseInfo, core.PrecedenceType, bool) {
	// Last match wins, so iterate in reverse.
	for i := len(t.Annotations) - 1; i >= 0; i-- {
		ann := &t.Annotations[i]
		if ann.Matches(path) {
			info := core.ReuseInfo{
				LicenseExpressions: ann.Licenses,
				CopyrightNotices:   ann.Copyrights,
				SourcePath:         t.Source,
				SourceType:         core.ReuseToml,
			}
			return info, ann.Precedence, true
		}
	}
	return core.ReuseInfo{}, core.Closest, false
}

func parsePrecedence(s string) (core.PrecedenceType, error) {
	switch s {
	case "", "closest":
		return core.Closest, nil
	case "aggregate":
		return core.Aggregate, nil
	case "override":
		return core.Override, nil
	default:
		return core.Closest, fmt.Errorf("unknown precedence %q", s)
	}
}
