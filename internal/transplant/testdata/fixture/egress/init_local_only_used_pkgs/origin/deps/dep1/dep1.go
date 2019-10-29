package dep1

import "origin.tld/user/proj/deps/dep2"

func ExportedFunc1() { // used by local
	dep2.ExportedFunc1()
}
