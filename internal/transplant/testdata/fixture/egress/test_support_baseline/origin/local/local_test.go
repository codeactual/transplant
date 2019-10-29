package local_test

import (
	"origin.tld/user/proj/dep1_only_used_by_local_test_file"
	"origin.tld/user/proj/local"
	"testing"
)

func TestExportedFunc1(t *testing.T) {
	local.ExportedFunc1()
	dep1_only_used_by_local_test_file.ExportedFunc1()
}
