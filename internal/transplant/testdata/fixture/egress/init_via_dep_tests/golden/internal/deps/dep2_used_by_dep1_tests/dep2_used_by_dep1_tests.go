package dep2_used_by_dep1_tests

import "copy.tld/user/proj/internal/deps/dep3_used_by_dep2_init"

func init() {
	dep3_used_by_dep2_init.Dep3Func()
}

func Dep2Func() {
}
