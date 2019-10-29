package proj

import "copy.tld/user/proj/internal/dep1"

func FromUnchanged1() {
	dep1.Dep1Func1()
}

func FromChanged1() {
	dep1.Dep1Func1()
}

func FromAdded1() {
	dep1.Dep1Func1()
}
