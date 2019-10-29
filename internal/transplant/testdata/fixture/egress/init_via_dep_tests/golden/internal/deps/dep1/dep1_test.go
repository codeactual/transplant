package dep1_test

import (
	"testing"

	"copy.tld/user/proj/internal/deps/dep2_used_by_dep1_tests"
)

func TestWithDependencyOnDep2(t *testing.T) {
	dep2_used_by_dep1_tests.Dep2Func()
}
