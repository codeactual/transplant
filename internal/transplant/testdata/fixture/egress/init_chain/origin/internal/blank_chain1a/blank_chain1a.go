package blank_chain1a

import (
	"origin.tld/user/proj/internal/blank_chain1a_init_dep"
	_ "origin.tld/user/proj/internal/blank_chain1b"
)

func init() {
	blank_chain1a_init_dep.Func()
}

// This package is only included for its init.
func UnusedShouldGetPruned() {
}
