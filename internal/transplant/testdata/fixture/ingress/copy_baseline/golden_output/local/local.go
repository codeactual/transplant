package local

import "origin.tld/user/proj/dep1"

// rewritten local path: origin.tld/user/proj/local/cpy_only
// rewritten dep path: origin.tld/user/proj/dep1
func FromUnchanged1() {
	dep1.Dep1Func1()
}

func FromChanged1() {
	dep1.Dep1Func1()
}

func FromAdded1() {
	dep1.Dep1Func1()
}
