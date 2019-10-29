package proj

import "copy.tld/user/proj/internal/dep1"

func ExportedFunc1() {
	_ = dep1.G1
}
