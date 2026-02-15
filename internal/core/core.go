// Package core holds the shared types used across reuse subpackages.
package core

// SourceType identifies where a piece of licensing information came from.
type SourceType int

const (
	FileHeader SourceType = iota
	DotLicense
	ReuseToml
	Dep5Source
)

func (s SourceType) String() string {
	switch s {
	case FileHeader:
		return "file-header"
	case DotLicense:
		return "dot-license"
	case ReuseToml:
		return "reuse-toml"
	case Dep5Source:
		return "dep5"
	default:
		return "unknown"
	}
}

// PrecedenceType controls how REUSE.toml annotations interact with in-file headers.
type PrecedenceType int

const (
	// Closest means the annotation applies only when the file itself has no
	// REUSE information. If the file has copyright but no license (or vice
	// versa), the missing piece is filled from the annotation.
	Closest PrecedenceType = iota

	// Aggregate means the annotation's info is combined with any in-file info.
	Aggregate

	// Override means the annotation replaces any in-file info entirely.
	// The file is not even read for REUSE tags.
	Override
)

func (p PrecedenceType) String() string {
	switch p {
	case Closest:
		return "closest"
	case Aggregate:
		return "aggregate"
	case Override:
		return "override"
	default:
		return "unknown"
	}
}

// ReuseInfo holds the licensing and copyright information extracted from a
// single source (file header, sidecar, REUSE.toml annotation, or dep5 paragraph).
type ReuseInfo struct {
	LicenseExpressions []string
	CopyrightNotices   []string
	Contributors       []string
	SourcePath         string
	SourceType         SourceType
}

// IsEmpty returns true if no license or copyright information is present.
func (r ReuseInfo) IsEmpty() bool {
	return len(r.LicenseExpressions) == 0 && len(r.CopyrightNotices) == 0
}

// HasLicense returns true if at least one license expression is present.
func (r ReuseInfo) HasLicense() bool {
	return len(r.LicenseExpressions) > 0
}

// HasCopyright returns true if at least one copyright notice is present.
func (r ReuseInfo) HasCopyright() bool {
	return len(r.CopyrightNotices) > 0
}
