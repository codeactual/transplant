package local

import (
	"copy.tld/user/proj/internal"
	"copy.tld/user/proj/internal/dep1"
)

func LocalFunc() {
	dep1.Dep1Func()
	_ = internal.Version
}
