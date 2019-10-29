package blank_chain0a

import (
	"origin.tld/user/proj/internal/blank_chain0a_init_dep"
	_ "origin.tld/user/proj/internal/blank_chain0b"
)

func init() {
	blank_chain0a_init_dep.Func()
}

// This package is only included for its init.
func UnusedShouldGetPruned() {
}
