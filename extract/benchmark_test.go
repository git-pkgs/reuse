package extract_test

import (
	"runtime"
	"strings"
	"testing"

	"github.com/git-pkgs/reuse/extract"
)

func BenchmarkExtractReuseInfo(b *testing.B) {
	for _, tc := range []struct {
		name string
		text string
	}{
		{"plain-1KiB", strings.Repeat("x", 1024)},
		{"plain-64KiB", strings.Repeat("x", 64<<10)},
		{"header-1KiB", "// SPDX-License-Identifier: MIT\n" + strings.Repeat("x", 1024)},
		{"header-64KiB", "// SPDX-License-Identifier: MIT\n" + strings.Repeat("x", 64<<10)},
		{"ignored", "REUSE-IgnoreStart\nSPDX-License-Identifier: GPL-3.0-only\nREUSE-IgnoreEnd\nSPDX-License-Identifier: MIT\n"},
		{"snippet", "SPDX-License-Identifier: MIT\nSPDX-SnippetBegin\nSPDX-License-Identifier: Apache-2.0\nSPDX-SnippetEnd\n"},
	} {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(tc.text)))
			for b.Loop() {
				_ = extract.ExtractReuseInfo(tc.text)
			}
		})
	}
}

func BenchmarkRetainedHeaders(b *testing.B) {
	const count = 64
	text := "// SPDX-License-Identifier: MIT\n" + strings.Repeat("x", 128<<10)
	for b.Loop() {
		values := make([]string, 0, count)
		var before, after runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&before)
		for range count {
			info := extract.ExtractReuseInfo(strings.Clone(text))
			values = append(values, info.LicenseExpressions...)
		}
		runtime.GC()
		runtime.ReadMemStats(&after)
		b.ReportMetric(float64(max(0, int64(after.HeapAlloc)-int64(before.HeapAlloc)))/count, "retained-B/file")
		runtime.KeepAlive(values)
	}
}
