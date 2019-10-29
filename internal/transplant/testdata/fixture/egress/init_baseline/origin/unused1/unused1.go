package unused1

import "origin.tld/user/proj/only_used_by_unused1"

func init() {
	only_used_by_unused1.ExportedFunc1()
}

func ExportedFunc1() {
}
