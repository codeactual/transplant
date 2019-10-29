package proj

import "copy.tld/user/proj/internal/dep1" // Import inline

func ExportedFunc1() {
	var i dep1.MyInt
	_ = i

	var s dep1.MyStruct
	s.Method1()

	dep1.ExportedFunc1()

	_ = dep1.S
}
