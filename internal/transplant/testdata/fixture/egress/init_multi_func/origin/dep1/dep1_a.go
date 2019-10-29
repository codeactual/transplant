package dep1

import "origin.tld/user/proj/dep2"

func init() {
	dep2.ExportedFunc1()
}

func ExportedFunc1() {
}
