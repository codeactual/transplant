package dep3

import "copy.tld/user/proj/internal/dep_four"

func ExportedFunc1() {
	dep_four.ExportedFunc1()
}
