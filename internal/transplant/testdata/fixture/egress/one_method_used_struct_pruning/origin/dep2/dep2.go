package dep2

import (
	"origin.tld/user/proj/dep3"
	"origin.tld/user/proj/unused1"
)

func ExportedFunc1() {
	nonExportedFunc1()
}

func nonExportedFunc1() {
	dep3.ExportedFunc1()
}

func nonExportedFunc2() { // unused
	unused1.ExportedFunc1()
}
