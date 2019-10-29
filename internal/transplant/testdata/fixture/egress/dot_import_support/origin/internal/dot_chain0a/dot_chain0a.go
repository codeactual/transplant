package dot_chain0a

import (
	"origin.tld/user/proj/internal/dot_chain0a_init_dep"
	. "origin.tld/user/proj/internal/dot_chain0b"
)

func init() {
	dot_chain0a_init_dep.Func()
}

func Chain0aFunc() {
	Chain0bFunc()
}
