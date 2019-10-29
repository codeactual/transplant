package dep2

import "copy.tld/user/proj/internal/dep3"

func ExportedFunc1() {
	dep3.ExportedFunc1()
}
