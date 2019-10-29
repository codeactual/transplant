package use

import (
	"fmt"

	other_pkg "fixture.tld/use_and_shadowing/pkglocal/var/use"
)

func init() {
	var a string = other_pkg.ExportedVar2
	_ = a
}

func InCall() {
	fmt.Sprint(other_pkg.ExportedVar2)
	fmt.Sprint(other_pkg.ExportedVar1)
}

func InSwitchCase() {
	var s string
	switch s {
	case other_pkg.ExportedVar2:
	case other_pkg.ExportedVar1:
	}
}

func InAssign() {
	var a string = other_pkg.ExportedVar2
	_ = a
}

func InAssignShort() {
	a := other_pkg.ExportedVar1
	b, c := other_pkg.ExportedVar2, other_pkg.ExportedVar3
	_, _, _ = a, b, c
}

func InCompositeLit() {
	type T struct {
		s string
	}
	t1 := T{s: other_pkg.ExportedVar2}

	t2 := struct {
		s string
	}{
		s: other_pkg.ExportedVar1,
	}

	_, _ = t1, t2
}

func InSingleReturn() string {
	return other_pkg.ExportedVar1
}

func InMultiReturn() (string, string) {
	return other_pkg.ExportedVar1, other_pkg.ExportedVar2
}

func InIf() {
	if other_pkg.ExportedVar1 == "" {
	}
	if other_pkg.ExportedVar2 == other_pkg.ExportedVar3 {
	}
}

func InSwitch() {
	switch other_pkg.ExportedVar1 {
	}
}
