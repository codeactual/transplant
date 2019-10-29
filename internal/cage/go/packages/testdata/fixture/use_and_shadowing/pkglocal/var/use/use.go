package use

import "fmt"

var (
	// these cannot be "" for use in switch cases below
	nonExportedVar1 string = "nonExportedVar1"
	ExportedVar1    string = "ExportedVar1"
)

type strType1 string

var (
	strVar1 strType1 = ""

	// assert that const-iota related logic is not applied to variables, where we assume
	// the first explicit type should be applied to the subsequent declarations which
	// lack an explicit type
	strVar2                = ""
	intWithoutExplicitType = 4
)

var ExportedVar2, ExportedVar3 string

func init() {
	var a string = nonExportedVar1
	_ = a
}

func InCall() {
	fmt.Sprint(nonExportedVar1)
	fmt.Sprint(ExportedVar1)
}

func InSwitchCase() {
	var s string
	switch s {
	case nonExportedVar1:
	case ExportedVar1:
	}
}

func InAssign() {
	var a string = nonExportedVar1
	_ = a
}

func InAssignShort() {
	a := ExportedVar1
	b, c := ExportedVar2, ExportedVar3
	_, _, _ = a, b, c
}

func InCompositeLit() {
	type T struct {
		s string
	}
	t1 := T{s: nonExportedVar1}

	t2 := struct {
		s string
	}{
		s: ExportedVar1,
	}

	_, _ = t1, t2
}

func InSingleReturn() string {
	return ExportedVar1
}

func InMultiReturn() (string, string) {
	return ExportedVar1, ExportedVar2
}

func InIf() {
	if ExportedVar1 == "" {
	}
	if ExportedVar2 == ExportedVar3 {
	}
}

func InSwitch() {
	switch ExportedVar1 {
	}
}
