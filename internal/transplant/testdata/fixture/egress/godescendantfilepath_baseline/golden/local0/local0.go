package local0

import (
	"copy.tld/user/proj/internal/dep1"
	"copy.tld/user/proj/local1"
)

func ExportedFunc1() {
	local1.ExportedFunc1()
	dep1.ExportedFunc1()
}
