package proj

import (
	"copy.tld/user/proj/internal/dep1"
	"copy.tld/user/proj/internal/dep2"
)

func ExportedFunc1() {
	dep1.ExportedFunc1()
	dep2.ExportedFunc1()
}
