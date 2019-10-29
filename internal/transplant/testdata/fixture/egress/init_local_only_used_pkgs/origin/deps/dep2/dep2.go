package dep2

import "origin.tld/user/proj/deps/dep3"

func ExportedFunc1() { // used by dep1
}

func ExportedFunc2() { // not used, so dep3's init should not be included
	dep3.ExportedFunc1()
}
