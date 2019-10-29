package dep1

import "origin.tld/user/proj/dep3"

func ExportedFunc1() {
	dep3.ExportedFunc1()
}
