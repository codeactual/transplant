package local1

import "origin.tld/user/proj/dep1"

func ExportedFunc1() {
	dep1.ExportedFunc1()
}
