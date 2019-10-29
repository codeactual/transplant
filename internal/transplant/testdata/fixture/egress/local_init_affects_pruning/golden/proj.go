package proj

import (
	"copy.tld/user/proj/internal/dep1"
)

func init() {
	dep1.ExportedFunc1()
}

func ExportedFunc1() {
	dep1.ExportedFunc2()
}
