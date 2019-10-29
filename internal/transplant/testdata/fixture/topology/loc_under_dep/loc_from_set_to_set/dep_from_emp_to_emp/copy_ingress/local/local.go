package local

import (
	"copy.tld/user/proj"
	"copy.tld/user/proj/dep1"
)

func LocalFunc() {
	dep1.Dep1Func()
	_ = proj.Version
}
