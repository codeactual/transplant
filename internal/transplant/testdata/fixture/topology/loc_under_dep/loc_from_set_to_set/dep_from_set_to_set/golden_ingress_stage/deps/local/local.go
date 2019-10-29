package local

import (
	"origin.tld/user/proj/deps/dep1"
	"origin.tld/user/proj/deps/local/local1"
)

func FromFunc() {
	local1.Local1Func()
	dep1.Dep1Func()
}
