// Package reuse extracts SPDX license and copyright information from projects
// following the REUSE specification v3.3 (https://reuse.software/spec-3.3/).
//
// It parses SPDX headers in source files, .license sidecar files, REUSE.toml
// annotations, and .reuse/dep5 files. SPDX expressions are stored as raw
// strings; validation is left to consumers.
package reuse

import "github.com/git-pkgs/reuse/internal/core"

// Type aliases re-exported from internal/core so consumers can use them
// as reuse.ReuseInfo, reuse.SourceType, etc.
type ReuseInfo = core.ReuseInfo
type SourceType = core.SourceType
type PrecedenceType = core.PrecedenceType

const (
	FileHeader = core.FileHeader
	DotLicense = core.DotLicense
	ReuseToml  = core.ReuseToml
	Dep5Source = core.Dep5Source

	Closest   = core.Closest
	Aggregate = core.Aggregate
	Override  = core.Override
)
