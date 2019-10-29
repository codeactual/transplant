package dep2

import "origin.tld/user/proj/dep4/dep4a"

func ExportedFunc1() {
	dep4a.ExportedFunc1()
}

func ExportedFunc2() {
}
