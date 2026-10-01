package extract_test

import (
	"fmt"

	"github.com/git-pkgs/reuse/extract"
)

func ExampleExtractReuseInfo() {
	info := extract.ExtractReuseInfo(`// SPDX-License-Identifier: MIT OR Apache-2.0
// SPDX-FileCopyrightText: 2024 Alice`)
	fmt.Println(info.LicenseExpressions)
	fmt.Println(info.CopyrightNotices)
	// Output:
	// [MIT OR Apache-2.0]
	// [2024 Alice]
}
