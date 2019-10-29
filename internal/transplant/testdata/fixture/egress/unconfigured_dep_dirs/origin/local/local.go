package local

import (
	"origin.tld/user/proj/dep1"
	"origin.tld/user/proj/dep_without_inclusion2"
)

func FromFunc() {
	dep1.Dep1Func()
	dep_without_inclusion2.DepWithoutExclusion2Func()
}
