package dep2

import "origin.tld/user/proj/dep2/dep2a"

func ExportedFunc1() {
	dep2a.ExportedFunc1()
}
