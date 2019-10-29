package dep2

import "origin.tld/user/proj/dep4"

func ExportedFunc1() {
	dep4.ExportedFunc1()
}
