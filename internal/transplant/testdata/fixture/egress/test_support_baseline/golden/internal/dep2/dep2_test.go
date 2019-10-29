package dep2_test

import (
	"copy.tld/user/proj/internal/dep2"
	"copy.tld/user/proj/internal/dep7"
	"testing"
)

func TestExportedFunc1(t *testing.T) {
	dep2.ExportedFunc1()
	dep7.ExportedFunc1()
}
