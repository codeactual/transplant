package dep1

import "origin.tld/user/proj/dep3"

func init() {
	dep3.ExportedFunc1()
}

func ExportedFunc2() { // should get pruned
}
