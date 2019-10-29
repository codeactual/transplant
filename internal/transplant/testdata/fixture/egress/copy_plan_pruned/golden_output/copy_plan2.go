package proj

import (
	"copy.tld/user/proj/internal/dep3"
	"copy.tld/user/proj/internal/dep_four"
	"copy.tld/user/proj/local1"
)

func ExportedFunc2() {
	dep3.ExportedFunc1()
	dep_four.ExportedFunc1()
	local1.ExportedFunc1()
}
