package local

import (
	"origin.tld/user/proj/dep3"
	"origin.tld/user/proj/dep4"
	"origin.tld/user/proj/local/local1"
)

func ExportedFunc2() {
	dep3.ExportedFunc1()
	dep4.ExportedFunc1()
	local1.ExportedFunc1()
}
