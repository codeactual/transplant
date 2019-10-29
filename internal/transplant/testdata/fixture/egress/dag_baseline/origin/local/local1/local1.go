package local1

import "origin.tld/user/proj/dep1"

func FuncCallChain() {
	dep1.FuncCallChain()
}

func MethodCallChainFromCompositeLit() {
	dep1.ExportedType1{}.MethodCallChainFromCompositeLit()
}

func MethodCallChainFromValue() {
	var v dep1.ExportedType1
	v.MethodCallChainFromValue()
}

func ConstChain() {
	_ = dep1.ConstChain
}
