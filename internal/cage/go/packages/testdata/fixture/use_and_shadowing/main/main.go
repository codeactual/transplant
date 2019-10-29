package main

import (
	const_pkg "fixture.tld/use_and_shadowing/pkglocal/const/use"
	func_pkg "fixture.tld/use_and_shadowing/pkglocal/func/use"
	type_pkg "fixture.tld/use_and_shadowing/pkglocal/type/use"
	var_pkg "fixture.tld/use_and_shadowing/pkglocal/var/use"
)

// For shadow.go
const nonExportedConst1 = ""
const nonExportedConst2 = ""
const ExportedConst3 = ""

const ExportedConst1 = ""
const ExportedConst2 = ""

type ExportedType1 struct{}

var ExportedVar1 string

func ExportedFunc1() {}

func init() {
	_ = ExportedConst1
	_ = const_pkg.ExportedConst2
}

func main() {
	var t1 ExportedType1
	var t2 type_pkg.ExportedType2

	ExportedFunc1()
	func_pkg.ExportedFunc1()

	_, _, _, _, _, _ = t1, t2, ExportedVar1, var_pkg.ExportedVar1, ExportedConst1, const_pkg.ExportedConst1
}
