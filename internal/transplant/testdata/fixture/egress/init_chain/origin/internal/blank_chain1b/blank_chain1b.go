package blank_chain1b

import (
	"origin.tld/user/proj/internal/blank_chain1b_init_dep"
)

func init() {
	blank_chain1b_init_dep.Func()
}

// This package is only included for its init.
func UnusedShouldGetPruned() {
}
