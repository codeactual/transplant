package use

import (
	"fmt"

	other_pkg "fixture.tld/use_and_shadowing/pkglocal/const/use"
)

const (
	ExportedConst4 = other_pkg.ExportedConst2
	ExportedConst5 = other_pkg.ExportedConst3 + "ExportedConst5" + other_pkg.ExportedConst2
)

const ExportedConst6, ExportedConst7 = other_pkg.ExportedConst4, other_pkg.ExportedConst5

func init() {
	var a string = other_pkg.ExportedConst2
	_ = a
}

func InCall() {
	fmt.Sprint(other_pkg.ExportedConst2)
	fmt.Sprint(other_pkg.ExportedConst1)
}

func InSwitchCase() {
	var s string
	switch s {
	case other_pkg.ExportedConst2:
	case other_pkg.ExportedConst1:
	}
}

func InAssign() {
	var a string = other_pkg.ExportedConst2
	_ = a
}

func InAssignShort() {
	a := other_pkg.ExportedConst1
	b, c := other_pkg.ExportedConst2, other_pkg.ExportedConst3
	_, _, _ = a, b, c
}

func InCompositeLit() {
	type T struct {
		s string
	}
	t1 := T{s: other_pkg.ExportedConst2}

	t2 := struct {
		s string
	}{
		s: other_pkg.ExportedConst1,
	}

	_, _ = t1, t2
}

func InSingleReturn() string {
	return other_pkg.ExportedConst1
}

func InMultiReturn() (string, string) {
	return other_pkg.ExportedConst1, other_pkg.ExportedConst2
}

func InLocalConst() {
	const (
		a = other_pkg.ExportedConst2
		b = other_pkg.ExportedConst1
	)
}

func InIf() {
	var a string
	if a == other_pkg.ExportedConst1 {
	}
}

func InSwitch() {
	var a string
	switch a + other_pkg.ExportedConst1 {
	}
}

func UseDefinedIntNonIota() {
	_, _, _ = other_pkg.DefinedIntNonIota0, other_pkg.DefinedIntNonIota1, other_pkg.DefinedIntNonIota2
}

func UseDefinedIntIotaInImportedConst() {
	if other_pkg.DefinedIntIota0 == 0 {
	}
	if other_pkg.DefinedIntIota1 == 0 {
	}
	if other_pkg.DefinedIntIota2 == 0 {
	}
}

const (
	DefinedIntIota0 other_pkg.DefinedInt = iota
	DefinedIntIota1
	DefinedIntIota2
)

func UseDefinedIntIotaInLocalConst() {
	_, _, _ = DefinedIntIota0, DefinedIntIota1, DefinedIntIota2
}
