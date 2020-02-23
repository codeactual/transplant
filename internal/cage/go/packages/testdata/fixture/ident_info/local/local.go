// In this fixture, all underlying types are defined locally or built-in.
package local

type Shadowed0 int
type Shadowed1 int

type nonExportedStruct0 struct {
	IntField int
}
type nonExportedStruct1 struct{}

// ExportedStruct0 asserts support for struct types and field/embedded types.
type ExportedStruct0 struct {
	nonExportedStruct0
	Field0 nonExportedStruct1
	Field1 func(DefinedInt) func(DefinedFunc)

	// assert var names not interpreted as type of same name

	Field2    func(Shadowed0 DefinedInt) (Shadowed1 DefinedInt)
	Shadowed0 DefinedInt
}

func (recv ExportedStruct0) Method0(p0 DefinedInt) (r0 AliasedInt) {
	return 0
}

type ExportedStruct1 struct {
}

// nonExportedDefinedInt/anotherNonExportedDefinedInt assert that type chains
// include non-exported globals.
type (
	nonExportedDefinedInt        int
	anotherNonExportedDefinedInt nonExportedDefinedInt
)

// DefinedInt is the primary node for asserting support for defined types.
type DefinedInt anotherNonExportedDefinedInt

// DefinedIntIota* assert support for value identifiers (and additionally those in iota groups).
const (
	DefinedIntIota0 DefinedInt = iota
	DefinedIntIota1
)

// nonExportedAliasedInt/anotherNonExportedAliasedInt assert that type chains
// include non-exported globals.
type (
	nonExportedAliasedInt        = int
	anotherNonExportedAliasedInt = nonExportedAliasedInt
)

// AliasedInt is the primary node for asserting support for aliased types.
type AliasedInt = anotherNonExportedAliasedInt

type definedFunc func(DefinedInt, AliasedInt) func(DefinedInt, AliasedInt)

type DefinedFunc definedFunc

type aliasedFunc = func(DefinedInt, AliasedInt) func(DefinedInt, AliasedInt)

type AliasedFunc = aliasedFunc

type LocalInterface0 interface {
	Method0(DefinedInt)
}

type LocalInterface0Impl0 struct{}

func (recv LocalInterface0Impl0) Method0(p0 DefinedInt) {
}

const IntLiteral100 = 100

func LocalFunc0(p0 DefinedInt, p1 AliasedInt) func(DefinedInt, AliasedInt) (r0 DefinedFunc, r1 AliasedFunc) {
	return nil
}

func LocalFunc1(p0, p1 DefinedInt, p2 ...AliasedInt) (r0, r1 DefinedInt, r2, r3 AliasedInt) {
	return 0, 0, 0, 0
}

func LocalFunc2(p0 func(DefinedInt) func(AliasedInt), p1 func(DefinedFunc) func(AliasedFunc)) func(nonExportedStruct0) func(nonExportedStruct1) {
	return nil
}

// LocalNonGlobalUse hosts fixtures for asserting support for identifiers outside global scope.
//
// (Its name is prefixed with "Local," unlike in dot.go/named.go, to avoid conflicting with the
// equivalent function in dot.go.)
func LocalNonGlobalUse() {
	_, _, _, _, _, _, _, _, _, _, _, _, _ = ExportedStruct0{
		nonExportedStruct0: nonExportedStruct0{},
		Field0:             nonExportedStruct1{},
		Shadowed0:          0,
	},
		ExportedStruct0{}.IntField, // from embdded nonExportedStruct0
		ExportedStruct0{}.Field0,
		ExportedStruct0{}.Field1,
		ExportedStruct0{}.Method0,
		ExportedStruct0{}.Method0(0),
		ExportedStruct0{}.Shadowed0,
		DefinedInt(6),
		AliasedInt(7),
		DefinedIntIota0,
		IntLiteral100,
		LocalFunc0,
		LocalFunc0(0, 0)

	var es0 ExportedStruct0
	_, _, _ = es0.IntField, es0.Field0, es0.Field1 // IntField from embedded nonExportedStruct0
	es0.Method0(0)

	var Shadowed0 DefinedInt
	_ = Shadowed0
}

// (Its name is prefixed with "Local," unlike in dot.go/named.go, to avoid conflicting with the
// equivalent function in dot.go.)
func LocalScopeTest() {
	// These identifiers should yield no IdentInfo because they refer to non-global names.

	var DefinedInt int
	_ = DefinedInt

	type ReDefinedInt int
	_ = ReDefinedInt(0)

	const DefinedIntIota0 = 0
	_ = DefinedIntIota0
}
