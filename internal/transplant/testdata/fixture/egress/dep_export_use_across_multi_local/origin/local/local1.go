package local

import "origin.tld/user/proj/dep1"

func FuncValue() {
	_ = dep1.ExportedFunc1
}

func FuncReturn() {
	_ = dep1.ExportedFunc2()
}
