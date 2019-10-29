package local

import (
	"origin.tld/user/proj/dep1"
	"origin.tld/user/proj/dep2"
)

func UseDep1() {
	dep1.ExportedFunc1()
}

func UseDep2() {
	dep2.ExportedFunc1()
}
