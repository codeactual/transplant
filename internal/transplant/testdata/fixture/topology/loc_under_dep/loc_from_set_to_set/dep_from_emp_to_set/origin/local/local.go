package local

import (
	"origin.tld/user/proj"
	"origin.tld/user/proj/dep1"
)

func LocalFunc() {
	dep1.Dep1Func()
	_ = proj.Version
}
