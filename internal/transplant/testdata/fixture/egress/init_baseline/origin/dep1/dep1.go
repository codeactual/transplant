package dep1

import (
	"origin.tld/user/proj/only_used_by_dep1_init"
)

func init() {
	only_used_by_dep1_init.ExportedFunc1()
}

func ExportedFunc1() {
}
