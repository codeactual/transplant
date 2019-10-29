package local

import (
	"origin.tld/user/proj/dep1"
)

func init() {
	dep1.ExportedFunc1()
}

func ExportedFunc1() {
	dep1.ExportedFunc2()
}
