package local

import (
	"origin.tld/user/proj/dep1"
	"origin.tld/user/proj/dep2"
)

func ExportedFunc1() {
	dep1.UsedFunc1()
	dep2.UsedFunc1()
}
