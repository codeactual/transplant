package dep1

import "origin.tld/user/proj/dep1/dep1a"

func ExportedFunc1() {
	dep1a.ExportedFunc1()
}
