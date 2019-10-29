package dep2

import "copy.tld/user/proj/internal/dep_four/dep4a"

func ExportedFunc1() {
	dep4a.ExportedFunc1()
}
