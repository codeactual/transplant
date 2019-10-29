package dep4_unused_test

import (
	"origin.tld/user/proj/dep4_unused"
	"origin.tld/user/proj/dep6_only_used_by_dep4_test_file"
	"testing"
)

func TestExportedFunc1(t *testing.T) {
	dep4_unused.ExportedFunc1()
	dep6_only_used_by_dep4_test_file.ExportedFunc1()
}
