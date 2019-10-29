package dep1

import "copy.tld/user/proj/internal/dep3"

func init() {
	dep3.ExportedFunc1()
}
