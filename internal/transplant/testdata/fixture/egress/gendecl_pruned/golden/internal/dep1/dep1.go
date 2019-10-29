package dep1

import (
	// runtime heading
	"runtime" // runtime inline
)

const (
	// ExportedConst1 heading
	ExportedConst1 = "Const1" // ExportedConst1 inline
)

// ExportedConst{4,5} heading
const ExportedConst4 = "Const4" // ExportedConst{4,5} inline

var (
	// ExportedVar1 heading
	ExportedVar1 string = "Var1" // ExportedVar1 inline
)

// ExportedVar{4,5} heading
var ExportedVar5 = "Var5" // ExportedVar{4,5} inline

type (
	// ExportedType2 heading
	ExportedType2 struct{ F2 string } // ExportedType2 inline
)

// ExportedType2Method1 heading
func (t ExportedType2) Method1() { // ExportedType2Method1 inline
	_ = runtime.GOOS
}

// ExportedFunc2 heading
func ExportedFunc2() { // ExportedFunc2 inline
	_ = runtime.GOOS
}
