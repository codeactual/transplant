package dep1

import (
	"bytes"
	"encoding/base64"
	"strings"

	// should not interfere with "runtime" above getting pruned, even though they share a package name
	my_runtime "copy.tld/user/proj/internal/dep1/runtime"
)

func ExportedFunc1() {
	_ = strings.Split("", "")
	my_runtime.ExportedFunc1()

	// Ensure we only look at the names of imported packages, not simply the package proj
	// of all nodes. For example, encoder is type io.WriterCloser but the "io" import should
	// still get pruned because after UnusedFunc1 is pruned there are no other instances of it.
	encoder := base64.NewEncoder(base64.StdEncoding, &bytes.Buffer{})
	encoder.Write([]byte{})
}
