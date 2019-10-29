package local

import "origin.tld/user/proj/deps/dep1"

func ExportedFunc1() {
	dep1.ExportedFunc1()
}
