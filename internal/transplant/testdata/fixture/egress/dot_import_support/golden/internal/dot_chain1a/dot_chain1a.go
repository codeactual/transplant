package dot_chain1a

import (
	"copy.tld/user/proj/internal/dot_chain1a_init_dep"
	. "copy.tld/user/proj/internal/dot_chain1b"
)

func init() {
	dot_chain1a_init_dep.Func()
}

func Chain1aFunc() {
	Chain1bFunc()
}
