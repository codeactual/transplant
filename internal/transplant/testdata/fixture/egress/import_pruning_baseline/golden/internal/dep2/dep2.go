package dep2

import (
	"go/build"
)

func ExportedFunc1() {
	_ = build.Default
}
