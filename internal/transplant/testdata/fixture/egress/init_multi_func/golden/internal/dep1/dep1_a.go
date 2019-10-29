package dep1

import "copy.tld/user/proj/internal/dep2"

func init() {
	dep2.ExportedFunc1()
}

func ExportedFunc1() {
}
