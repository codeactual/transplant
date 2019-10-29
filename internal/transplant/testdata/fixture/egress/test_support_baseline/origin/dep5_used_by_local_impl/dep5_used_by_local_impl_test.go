package dep5_used_by_local_impl_test

import (
	"origin.tld/user/proj/dep5_used_by_local_impl"
	"testing"
)

func TestExportedFunc1(t *testing.T) {
	dep5_used_by_local_impl.ExportedFunc1()
}
