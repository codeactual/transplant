package dep1_test

import (
	"testing"

	"origin.tld/user/proj/deps/dep2_used_by_dep1_tests"
)

func TestWithDependencyOnDep2(t *testing.T) {
	dep2_used_by_dep1_tests.Dep2Func()
}
