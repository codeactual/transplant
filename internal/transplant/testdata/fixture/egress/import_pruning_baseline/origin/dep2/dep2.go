package dep2

import (
	"go/build"
	"io/ioutil" // should get pruned
)

func ExportedFunc1() {
	_ = build.Default
}

func UnusedFunc1() {
	_ = ioutil.Discard
}
