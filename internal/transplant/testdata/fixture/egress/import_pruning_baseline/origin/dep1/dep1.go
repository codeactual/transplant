package dep1

import (
	"bytes"
	"encoding/base64"
	"io"      // should get pruned
	"runtime" // should get pruned
	"strings"

	// should not interfere with "runtime" above getting pruned, even though they share a package name
	my_runtime "origin.tld/user/proj/dep1/runtime"

	my_ioutil "io/ioutil" // should get pruned
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

func UnusedFunc1() {
	_ = runtime.GOOS
	_ = io.EOF

	// Ensure we do not consider the "my_ioutil" ast.Ident in the import statement as use of the "io/ioutil" package
	// and instead only look at nodes outside the import.
	_ = my_ioutil.Discard
}
