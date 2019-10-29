package local0

import (
	"origin.tld/user/proj/dep1"
	"origin.tld/user/proj/local/local1"
)

func ExportedFunc1() {
	local1.ExportedFunc1()
	dep1.ExportedFunc1()
}
