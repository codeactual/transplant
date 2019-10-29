package local

import "origin.tld/user/proj/dep2"

func ExportedFunc1() {
	dep2.ExportedFunc1()
}
