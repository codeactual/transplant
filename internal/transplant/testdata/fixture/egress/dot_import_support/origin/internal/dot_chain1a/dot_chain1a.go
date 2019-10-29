package dot_chain1a

import (
	"origin.tld/user/proj/internal/dot_chain1a_init_dep"
	. "origin.tld/user/proj/internal/dot_chain1b"
)

func init() {
	dot_chain1a_init_dep.Func()
}

func Chain1aFunc() {
	Chain1bFunc()
}
