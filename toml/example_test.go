package toml_test

import (
	"fmt"

	"github.com/git-pkgs/reuse/toml"
)

func ExampleParseReuseTOML() {
	config, err := toml.ParseReuseTOML(`version = 1
[[annotations]]
path = "src/**"
SPDX-License-Identifier = "MIT"
`)
	if err != nil {
		fmt.Println(err)
		return
	}
	info, precedence, ok := config.ReuseInfoOf("src/main.go")
	if ok {
		fmt.Println(info.LicenseExpressions)
		fmt.Println(precedence)
	}
	// Output:
	// [MIT]
	// closest
}

func ExampleParseReuseTOMLFile() {
	config, err := toml.ParseReuseTOMLFile("../testdata/fake_repository/REUSE.toml")
	if err != nil {
		fmt.Println(err)
		return
	}
	info, precedence, ok := config.ReuseInfoOf("src/main.go")
	if ok {
		fmt.Println(info.LicenseExpressions)
		fmt.Println(precedence)
	}
	// Output:
	// [MIT]
	// closest
}
