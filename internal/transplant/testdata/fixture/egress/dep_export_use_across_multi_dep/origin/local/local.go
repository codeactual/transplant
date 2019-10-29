package local

import (
	"origin.tld/user/proj/dep1"
	"origin.tld/user/proj/dep2"
)

func MethodValue() {
	_ = dep2.ExportedType1{}.Method1
}

func MethodReturn() {
	_ = dep2.ExportedType2{}.Method1()
}

func FuncValue() {
	_ = dep1.ExportedFunc1
}

func FuncReturn() {
	_ = dep1.ExportedFunc2()
}

func StructFieldType() {
	type t struct {
		v dep2.ExportedType1
	}
}
