package dep1

import "origin.tld/user/proj/dep2"

type ExportedType1 struct {
}

func (t ExportedType1) UnusedMethod1() {
	dep2.ExportedFunc1()
}
