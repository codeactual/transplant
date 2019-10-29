package baseline

import (
	"runtime"
	"strings"

	"fixture.tld/packages_used_by_node/baseline/dep1"
	"fixture.tld/packages_used_by_node/baseline/dep2"

	// Don't let goimports name the import accurately so we can verify the detected
	// name comes from a source that uses the package clause instead of the import statement.
	"fixture.tld/packages_used_by_node/baseline/pkg_name_differs"
)

func ExportedFunc1() {
	_ = strings.TrimSpace("")
	dep1.ExportedFunc1()
}

func ExportedFunc2() {
	_ = runtime.GOOS
	dep2.ExportedFunc1()
}

func ExportedFunc3() {
	pkg_name_differs_from_dir_name.ExportedFunc1()
}
