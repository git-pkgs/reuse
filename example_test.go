package reuse_test

import (
	"fmt"

	"github.com/git-pkgs/reuse"
)

func ExampleProject_ReuseInfoOf() {
	project, err := reuse.OpenProject("testdata/fake_repository")
	if err != nil {
		fmt.Println(err)
		return
	}
	info, err := project.ReuseInfoOf("src/main.go")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(info.LicenseExpressions)
	fmt.Println(info.CopyrightNotices)
	// Output:
	// [MIT]
	// [2024 Alice]
}
