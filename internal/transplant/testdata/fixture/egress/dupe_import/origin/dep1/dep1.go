package dep1

import (
	"runtime"
	rt "runtime"
)

func Dep1Func() {
	_ = runtime.GOOS
	_ = rt.GOOS
}
