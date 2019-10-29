package dep2

import "origin.tld/user/proj/dep3"

const ConstChain = dep3.ConstChain

type NamedTypeChain dep3.NamedTypeChain

type AliasTypeChain = dep3.AliasTypeChain

type FieldTypeChain struct {
	f dep3.FieldTypeChain
}

type ExportedType1 struct{}

func (t ExportedType1) MethodCallChainFromCompositeLit() {
	dep3.ExportedType1{}.MethodCallChainFromCompositeLit()
}

func (t ExportedType1) MethodCallChainFromValue() {
	var v dep3.ExportedType1
	v.MethodCallChainFromValue()
}

func FuncCallChain() {
	dep3.FuncCallChain()
}
