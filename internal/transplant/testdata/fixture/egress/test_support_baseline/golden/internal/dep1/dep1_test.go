package dep1_test

import (
	"copy.tld/user/proj/internal/dep1"
	"copy.tld/user/proj/internal/dep2"
	"testing"
)

func TestExportedFunc1(t *testing.T) {
	dep1.ExportedFunc1()
	dep2.ExportedFunc1()
}
