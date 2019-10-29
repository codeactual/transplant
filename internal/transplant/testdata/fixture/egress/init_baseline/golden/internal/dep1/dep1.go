package dep1

import (
	"copy.tld/user/proj/internal/only_used_by_dep1_init"
)

func init() {
	only_used_by_dep1_init.ExportedFunc1()
}

func ExportedFunc1() {
}
