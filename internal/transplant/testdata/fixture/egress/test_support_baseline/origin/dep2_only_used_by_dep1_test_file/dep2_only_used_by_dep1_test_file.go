package dep2_only_used_by_dep1_test_file

import "origin.tld/user/proj/dep3_only_used_by_dep2"

func ExportedFunc1() {
	dep3_only_used_by_dep2.ExportedFunc1()
}

func UnusedFunc1() {
}
