package shadow

import (
	// declared as "pkg_name_is_not_dir_name" in source file
	pkg_name_is_not_dir_name "fixture.tld/use_and_shadowing/pkglocal/import_name/shadow/pkg_name_differs"

	used_name_non_stdlib "fixture.tld/use_and_shadowing/pkglocal/import_name/shadow/non_stdlib"
	used_name_stdlib "fmt"
	"strings"
)

func init() {
	// ensure usage
	_ = strings.Join
	_ = used_name_stdlib.Sprintf
	_ = used_name_non_stdlib.ExportedConst1
	_ = pkg_name_is_not_dir_name.ExportedConst1
}

func FuncParamDeclName(strings string) {
}

func FuncParamUsedNameStdlib(used_name_stdlib string) {
}

func FuncParamUsedNameNonStdlib(used_name_non_stdlib string) {
}

func FuncParamDeclNameDiffersFromDir(pkg_name_is_not_dir_name string) {
}

type typeWithShadowParam struct {
}

type typeWithShadowReceiver struct {
}

func (strings typeWithShadowReceiver) DeclName() {
}

func (pkg_name_is_not_dir_name typeWithShadowReceiver) DeclNameDiffersFromDir() {
}

func (used_name_stdlib typeWithShadowReceiver) UsedNameStdlib() {
}

func (used_name_non_stdlib typeWithShadowReceiver) UsedNameNonStdlib() {
}

func ShadowShortVar() {
	strings := ""
	_ = strings

	used_name_stdlib := ""
	_ = used_name_stdlib

	used_name_non_stdlib := ""
	_ = used_name_non_stdlib

	pkg_name_is_not_dir_name := ""
	_ = pkg_name_is_not_dir_name
}

func ShadowLongVar() {
	var pkg_name_is_not_dir_name, used_name_stdlib, used_name_non_stdlib, strings string
	_, _, _, _ = pkg_name_is_not_dir_name, used_name_stdlib, used_name_non_stdlib, strings
}

func ShadowConst() {
	const (
		pkg_name_is_not_dir_name = ""
		used_name_stdlib         = ""
		used_name_non_stdlib     = ""
		strings                  = ""
	)
}

func ShadowType() {
	type pkg_name_is_not_dir_name struct{}
	type strings struct{}
	type used_name_stdlib struct{}
	type used_name_non_stdlib struct{}
}
