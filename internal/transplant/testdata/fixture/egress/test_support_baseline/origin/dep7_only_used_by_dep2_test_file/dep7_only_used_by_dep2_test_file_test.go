package dep7_only_used_by_dep2_test_file_test

import (
	"origin.tld/user/proj/dep7_only_used_by_dep2_test_file"
	"testing"
)

func TestExportedFunc1(t *testing.T) {
	dep7_only_used_by_dep2_test_file.ExportedFunc1()
}
