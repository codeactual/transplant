package init_chain1b_test

import (
	"testing"

	"origin.tld/user/proj/internal/init_chain1b"
	"origin.tld/user/proj/internal/init_chain1b_init_dep"
)

func init() {
	init_chain1b_init_dep.Func()
}

func TestFunc(t *testing.T) {
	init_chain1b.Func()
}
