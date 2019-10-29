package internal

import "golang.org/x/sync/errgroup"

func Dep1Func() {
	var g errgroup.Group
	_ = g
}
