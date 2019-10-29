package use

import "fmt"

const (
	// these cannot be "" for use in switch cases below
	nonExportedConst1 = "nonExportedConst1"
	ExportedConst1    = "ExportedConst1"

	ExportedConst2 = ""
	ExportedConst3 = ""

	ExportedConst4 = ExportedConst2
	ExportedConst5 = ExportedConst3 + "ExportedConst5" + ExportedConst2
)

const ExportedConst6, ExportedConst7 = ExportedConst4, ExportedConst5

func init() {
	var a string = nonExportedConst1
	_ = a
}

func InCall() {
	fmt.Sprint(nonExportedConst1)
	fmt.Sprint(ExportedConst1)
}

func InSwitchCase() {
	var s string
	switch s {
	case nonExportedConst1:
	case ExportedConst1:
	}
}

func InAssign() {
	var a string = nonExportedConst1
	_ = a
}

func InAssignShort() {
	a := ExportedConst1
	b, c := ExportedConst2, ExportedConst3
	_, _, _ = a, b, c
}

func InCompositeLit() {
	type T struct {
		s string
	}
	t1 := T{s: nonExportedConst1}

	t2 := struct {
		s string
	}{
		s: ExportedConst1,
	}

	_, _ = t1, t2
}

func InSingleReturn() string {
	return ExportedConst1
}

func InMultiReturn() (string, string) {
	return ExportedConst1, ExportedConst2
}

func InLocalConst() {
	const (
		a = nonExportedConst1
		b = ExportedConst1
	)
}

func InIf() {
	var a string
	if a == ExportedConst1 {
	}
}

func InSwitch() {
	var a string
	switch a + ExportedConst1 {
	}
}

type DefinedInt int

const (
	DefinedIntNonIota0 DefinedInt = 0
	DefinedIntNonIota1 DefinedInt = 1
	DefinedIntNonIota2 DefinedInt = 2
)

func UseDefinedIntNonIota() {
	_, _, _ = DefinedIntNonIota0, DefinedIntNonIota1, DefinedIntNonIota2
}

const (
	DefinedIntIota0 DefinedInt = iota
	DefinedIntIota1
	DefinedIntIota2
)

func UseDefinedIntIota() {
	_, _, _ = DefinedIntIota0, DefinedIntIota1, DefinedIntIota2
}

const (
	MultiValIota0, MultiValIota1 DefinedInt = iota * 4, iota + 4
	MultiValIota2, MultiValIota3

	// Inspector should not detect use of DefinedInt in these
	MultiValIota4, MultiValIota5 = iota * 4, iota + 4
	MultiValIota6, MultiValIota7
)

func UseMultiValIota() {
	_, _, _, _ = MultiValIota0, MultiValIota1, MultiValIota2, MultiValIota3
	_, _, _, _ = MultiValIota4, MultiValIota5, MultiValIota6, MultiValIota7
}
