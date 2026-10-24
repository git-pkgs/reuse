package dep5_test

import (
	"fmt"

	"github.com/git-pkgs/reuse/dep5"
)

func ExampleParseDep5() {
	config, err := dep5.ParseDep5(`Format: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/

Files: docs/*
Copyright: 2024 Alice
License: CC-BY-4.0
`)
	if err != nil {
		fmt.Println(err)
		return
	}
	info, ok := config.ReuseInfoOf("docs/guide.md")
	if ok {
		fmt.Println(info.LicenseExpressions)
	}
	// Output:
	// [CC-BY-4.0]
}
