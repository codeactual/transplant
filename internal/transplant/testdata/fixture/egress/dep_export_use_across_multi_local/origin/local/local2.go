package local

import "origin.tld/user/proj/dep1"

func MethodValue() {
	_ = dep1.ExportedType1{}.Method1
}

func MethodReturn() {
	_ = dep1.ExportedType2{}.Method1()
}

func StructFieldType() {
	type t struct {
		v dep1.ExportedType1
	}
}
