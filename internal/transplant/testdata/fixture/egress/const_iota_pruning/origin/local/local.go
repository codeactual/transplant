package local

import "origin.tld/user/proj/dep1"

func ExportedFunc1() {
	_ = dep1.G1
}
