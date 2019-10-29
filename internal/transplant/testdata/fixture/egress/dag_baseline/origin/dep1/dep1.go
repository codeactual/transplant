package dep1

import (
	"origin.tld/user/proj/dep2"
)

const ConstChain = dep2.ConstChain

type NamedTypeChain dep2.NamedTypeChain

type AliasTypeChain = dep2.AliasTypeChain

type FieldTypeChain struct {
	f dep2.FieldTypeChain
}

type ExportedType1 struct{}

func (t ExportedType1) MethodCallChainFromCompositeLit() {
	dep2.ExportedType1{}.MethodCallChainFromCompositeLit()
}

func (t ExportedType1) MethodCallChainFromValue() {
	var v dep2.ExportedType1
	v.MethodCallChainFromValue()
}

func FuncCallChain() {
	dep2.FuncCallChain()
}
