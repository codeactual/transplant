// In this fixture, all underlying types are defined locally, built-in, or from named imports.
package named

import (
	other_pkg0 "fixture.tld/ident_info/local"
	other_pkg1 "fixture.tld/ident_info/non_inspected"
)

// ExportedStruct0 asserts detection of embedded types of the same name.
type ExportedStruct0 struct {
	other_pkg0.ExportedStruct0
}

// ConstFromNonInspectedPkg assert lack of type info on ConstFromNonInspectedPkg
// and lack of IdentInfo about NonInspectedConst.
const ConstFromNonInspectedPkg = other_pkg1.NonInspectedConst

const (
	// CustomDefinedIntIota* assert support for value identifiers (and additionally those in iota groups)
	// where the type is dot-imported.
	CustomDefinedIntIota0 other_pkg0.DefinedInt = iota
	CustomDefinedIntIota1
)

// DefinedIntIota0 assists with the assertion that an imported constant of the same name
// is not mistaken for this one.
const DefinedIntIota0 = 0

// DefinedIntIota0Copy assert support for assignments of dot-imported values on the right-hand side.
const DefinedIntIota0Copy = other_pkg0.DefinedIntIota0

// ExportedStruct0 asserts support for struct types and and field/embedded types which are dot-imported.
type CustomExportedStruct0 struct {
	other_pkg0.ExportedStruct0
	Field0 other_pkg0.ExportedStruct1
}

func (recv CustomExportedStruct0) Method0(p0 other_pkg0.DefinedInt) (r0 other_pkg0.AliasedInt) {
	return 0
}

// DefinedInt asserts that, when used, the correct import path origin is determined.
type DefinedInt other_pkg0.DefinedInt

// ReDefinedInt* asserts support for defined-type chains where the initial type is dot-imported.

type ReDefinedInt other_pkg0.DefinedInt

type ReDefinedIntTwice ReDefinedInt

type ReDefinedIntThrice ReDefinedIntTwice

// ReAliasedInt* asserts support for aliased-type chains where the initial type is dot-imported.

type ReAliasedInt = other_pkg0.AliasedInt

type ReAliasedIntTwice = ReAliasedInt

type ReAliasedIntThrice = ReAliasedIntTwice

type ReDefinedFunc other_pkg0.DefinedFunc

type ReAliasedFunc = other_pkg0.AliasedFunc

type Interface0 interface {
	Method0(other_pkg0.DefinedInt)
}

type Interface1 interface {
	Method0(DefinedInt)
}

func NamedFunc0(p0 other_pkg0.DefinedInt, p1 other_pkg0.AliasedInt) func(other_pkg0.DefinedInt, other_pkg0.AliasedInt) (r0 other_pkg0.DefinedFunc, r1 other_pkg0.AliasedFunc) {
	return nil
}

// NonGlobalUse hosts fixtures for asserting support for identifiers outside global scope.
func NonGlobalUse() {
	// Assert use of globals defined above in this file.

	_, _, _, _, _, _ = CustomExportedStruct0{
		ExportedStruct0: other_pkg0.ExportedStruct0{},
		Field0:          other_pkg0.ExportedStruct1{},
	},
		other_pkg0.ExportedStruct0{}.Method0,
		other_pkg0.ExportedStruct0{}.Method0(0),
		CustomDefinedIntIota0,
		ReDefinedIntThrice(4),
		ReAliasedIntThrice(5)

	var es0 other_pkg0.ExportedStruct0
	es0.Method0(0)

	// Assert use of globals name-imported into this file.
	_, _, _ =
		other_pkg0.DefinedInt(6),
		other_pkg0.AliasedInt(7),
		other_pkg0.DefinedIntIota0
}

func ScopeTest() {
	// Both DefinedInt identifiers should yield no IdentInfo because they refer to non-global names.

	var ReDefinedInt int
	_ = ReDefinedInt

	type ReAliasedInt int
	_ = ReAliasedInt(0)

	const DefinedIntIota0Copy = 0
	_ = DefinedIntIota0Copy
}

// Assert detection of the implementation's type and method.
func ImportedInterfaceImplUse() {
	other_pkg0.LocalInterface0Impl0{}.Method0(0)
}
