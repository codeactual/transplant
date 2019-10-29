package blank_chain0b

import (
	"origin.tld/user/proj/internal/blank_chain0b_init_dep"
)

func init() {
	blank_chain0b_init_dep.Func()
}

// This package is only included for its init.
func UnusedShouldGetPruned() {
}
