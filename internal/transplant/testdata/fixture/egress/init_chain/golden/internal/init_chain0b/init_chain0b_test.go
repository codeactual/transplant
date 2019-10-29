package init_chain0b_test

import (
	"testing"

	"copy.tld/user/proj/internal/init_chain0b"
	"copy.tld/user/proj/internal/init_chain0b_init_dep"
)

func init() {
	init_chain0b_init_dep.Func()
}

func TestFunc(t *testing.T) {
	init_chain0b.Func()
}
