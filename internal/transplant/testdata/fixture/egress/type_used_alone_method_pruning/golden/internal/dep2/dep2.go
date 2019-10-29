package dep2

import (
	"copy.tld/user/proj/internal/dep3"
)

func ExportedFunc1() {
	nonExportedFunc1()
}

func nonExportedFunc1() {
	dep3.ExportedFunc1()
}
