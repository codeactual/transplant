package use

import (
	other_pkg "fixture.tld/use_and_shadowing/pkglocal/func/use"
)

func init() {
	a := other_pkg.ExportedFunc2
	b := other_pkg.ExportedFunc1()
	_, _ = a, b
}

func InCallAsValue() {
	f := func(v func() string) {}
	f(other_pkg.ExportedFunc2)
}

func InCallForReturn() {
	f := func(v string) {}
	f(other_pkg.ExportedFunc1())
}

func InIfForReturn() {
	if other_pkg.ExportedFunc2() == "" {
	}
}

func InSwitchForReturn() {
	switch other_pkg.ExportedFunc1() {
	}
}

func InSwitchCaseForReturn() {
	var a string
	switch a {
	case other_pkg.ExportedFunc2():
	}
}

func InSelectCaseForReturn() {
	select {
	case <-other_pkg.ExportedFunc4():
	}
}

func InSelectCaseAssignForReturn() {
	select {
	case f := <-other_pkg.ExportedFunc4():
		_ = f
	}
}

func InAssignAsValue() {
	var a interface{} = other_pkg.ExportedFunc1
	_ = a
}

func InAssignShortAsValue() {
	a := other_pkg.ExportedFunc2
	b, c := other_pkg.ExportedFunc1, other_pkg.ExportedFunc3
	_, _, _ = a, b, c
}

func InAssignForReturn() {
	var a string = other_pkg.ExportedFunc2()
	_ = a
}

func InAssignShortForReturn() {
	a := other_pkg.ExportedFunc2()
	b, c := other_pkg.ExportedFunc1(), other_pkg.ExportedFunc3()
	_, _, _ = a, b, c
}

func InCompositeLitAsValue() {
	type T struct {
		f func() string
	}
	t1 := T{f: other_pkg.ExportedFunc2}

	t2 := struct {
		f func() string
	}{
		f: other_pkg.ExportedFunc1,
	}

	_, _ = t1, t2
}

func InCompositeLitForReturn() {
	type T struct {
		s string
	}
	t1 := T{s: other_pkg.ExportedFunc2()}

	t2 := struct {
		s string
	}{
		s: other_pkg.ExportedFunc1(),
	}

	_, _ = t1, t2
}

func InSingleReturnAsValue() func() string {
	return other_pkg.ExportedFunc1
}

func InMultiReturnAsValue() (func() string, func() string) {
	return other_pkg.ExportedFunc2, other_pkg.ExportedFunc1
}

func InSingleReturnForReturn() func() string {
	return other_pkg.ExportedFunc1
}

func InMultiReturnForReturn() (string, string) {
	return other_pkg.ExportedFunc2(), other_pkg.ExportedFunc1()
}

func InDefer() {
	defer other_pkg.ExportedFunc1()
}

func InGoroutine() {
	go other_pkg.ExportedFunc1()
}
