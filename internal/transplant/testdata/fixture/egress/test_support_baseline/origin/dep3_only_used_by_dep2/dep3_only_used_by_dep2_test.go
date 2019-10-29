package dep3_only_used_by_dep2_test

import (
	"origin.tld/user/proj/dep3_only_used_by_dep2"
	"testing"
)

func TestExportedFunc1(t *testing.T) {
	dep3_only_used_by_dep2.ExportedFunc1()
}
