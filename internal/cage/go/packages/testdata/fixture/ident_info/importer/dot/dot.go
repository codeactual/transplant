// In this fixture, all underlying types are defined locally, built-in, or from dot imports.
package dot

import (
	. "fixture.tld/ident_info/local"
	. "fixture.tld/ident_info/non_inspected"
)

// ConstFromNonInspectedPkg assert lack of type info on ConstFromNonInspectedPkg
// and lack of IdentInfo about NonInspectedConst.
const ConstFromNonInspectedPkg = NonInspectedConst

// CustomDefinedIntIota* assert support for value identifiers (and additionally those in iota groups)
// where the type is dot-imported.
const (
	CustomDefinedIntIota0 DefinedInt = iota
	CustomDefinedIntIota1
)

const (
	// DefinedIntIota0Copy assert support for assignments of dot-imported values on the right-hand side.
	DefinedIntIota0Copy = DefinedIntIota0
)

// CustomExportedStruct0 asserts support for struct types and and field/embedded types which are dot-imported.
type CustomExportedStruct0 struct {
	ExportedStruct0
	Field0 ExportedStruct1
	Next   *CustomExportedStruct0
}

func (recv CustomExportedStruct0) Method0(p0 DefinedInt) (r0 AliasedInt) {
	return 0
}

// ReDefinedInt* asserts support for defined-type chains where the initial type is dot-imported.

type ReDefinedInt DefinedInt

type ReDefinedIntTwice ReDefinedInt

type ReDefinedIntThrice ReDefinedIntTwice

// ReAliasedInt* asserts support for aliased-type chains where the initial type is dot-imported.

type ReAliasedInt = AliasedInt

type ReAliasedIntTwice = ReAliasedInt

type ReAliasedIntThrice = ReAliasedIntTwice

type ReDefinedFunc DefinedFunc

type ReAliasedFunc = AliasedFunc

type Interface0 interface {
	Method0(DefinedInt)
}

func Func0(p0 DefinedInt, p1 AliasedInt) func(DefinedInt, AliasedInt) (r0 DefinedFunc, r1 AliasedFunc) {
	return nil
}

// NonGlobalUse hosts fixtures for asserting support for identifiers outside global scope.
func NonGlobalUse() {
	// Assert use of globals defined above in this file.

	_, _, _, _, _, _, _, _ = CustomExportedStruct0{
		ExportedStruct0: ExportedStruct0{},
		Field0:          ExportedStruct1{},
	},
		ExportedStruct0{}.Method0,
		ExportedStruct0{}.Method0(0),
		CustomDefinedIntIota0,
		CustomDefinedIntIota1,
		ReDefinedIntThrice(4),
		ReAliasedIntThrice(5),
		DefinedIntIota0Copy

	var es0 ExportedStruct0
	es0.Method0(0)

	// Assert use of globals name-imported into this file.
	_, _, _ = DefinedInt(6),
		AliasedInt(7),
		DefinedIntIota0
}

func ScopeTest() {
	// Both DefinedInt identifiers should yield no IdentInfo because they refer to non-global names.

	var DefinedInt int
	_ = DefinedInt

	type ReDefinedInt int
	_ = ReDefinedInt(0)

	const DefinedIntIota0 = 0
	_ = DefinedIntIota0
}
