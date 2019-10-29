package proj_test

import (
	"copy.tld/user/proj"
	"copy.tld/user/proj/internal/dep1"
	"testing"
)

func TestExportedFunc1(t *testing.T) {
	proj.ExportedFunc1()
	dep1.ExportedFunc1()
}
