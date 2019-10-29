package proj

import "copy.tld/user/proj/internal/deps/dep1"

func ExportedFunc1() {
	dep1.ExportedFunc1()
}
