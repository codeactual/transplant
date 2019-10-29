package dep1

import "copy.tld/user/proj/internal/deps/dep2"

func ExportedFunc1() { // used by local
	dep2.ExportedFunc1()
}
