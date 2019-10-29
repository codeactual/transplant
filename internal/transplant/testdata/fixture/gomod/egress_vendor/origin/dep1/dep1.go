package dep1

import "golang.org/x/sync/errgroup"

func Dep1Func() {
	var g errgroup.Group
	_ = g
}
