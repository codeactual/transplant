package init_chain1b_test

import (
	"testing"

	"copy.tld/user/proj/internal/init_chain1b"
	"copy.tld/user/proj/internal/init_chain1b_init_dep"
)

func init() {
	init_chain1b_init_dep.Func()
}

func TestFunc(t *testing.T) {
	init_chain1b.Func()
}
