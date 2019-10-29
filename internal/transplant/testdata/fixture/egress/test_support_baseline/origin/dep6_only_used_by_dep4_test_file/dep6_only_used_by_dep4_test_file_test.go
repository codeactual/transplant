package dep6_only_used_by_dep4_test_file_test

import (
	"origin.tld/user/proj/dep6_only_used_by_dep4_test_file"
	"testing"
)

func TestExportedFunc1(t *testing.T) {
	dep6_only_used_by_dep4_test_file.ExportedFunc1()
}
