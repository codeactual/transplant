package proj

import "copy.tld/user/proj/internal/dep1"

func ExportedFunc1() {
	t := dep1.ExportedType2{}
	t.Method1()
	dep1.ExportedFunc2()
	_ = dep1.ExportedConst1
	_ = dep1.ExportedConst4
	_ = dep1.ExportedVar1
	_ = dep1.ExportedVar5
}
