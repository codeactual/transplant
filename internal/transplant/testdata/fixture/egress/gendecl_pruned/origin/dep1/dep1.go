package dep1

import (
	// strings heading
	"strings" // strings inline

	// runtime heading
	"runtime" // runtime inline

	// build heading
	"go/build" // build inline
)

const (
	// ExportedConst1 heading
	ExportedConst1 = "Const1" // ExportedConst1 inline

	// ExportedConst2 heading
	ExportedConst2 = "Const2" // ExportedConst2 inline

	// ExportedConst3 heading
	ExportedConst3 = "Const3" // ExportedConst3 inline
)

// ExportedConst{4,5} heading
const ExportedConst4, ExportedConst5 = "Const4", "Const5" // ExportedConst{4,5} inline

var (
	// ExportedVar1 heading
	ExportedVar1 string = "Var1" // ExportedVar1 inline

	// ExportedVar2 heading
	ExportedVar2 string = "Var2" // ExportedVar2 inline

	// ExportedVar3 heading
	ExportedVar3 string = "Var3" // ExportedVar3 inline
)

// ExportedVar{4,5} heading
var ExportedVar4, ExportedVar5 = "Var4", "Var5" // ExportedVar{4,5} inline

type (
	// ExportedType1 heading
	ExportedType1 struct{ F1 string } // ExportedType1 inline

	// ExportedType2 heading
	ExportedType2 struct{ F2 string } // ExportedType2 inline

	// ExportedType3 heading
	ExportedType3 struct{ F2 string } // ExportedType3 inline
)

// ExportedType1Method1 heading
func (t ExportedType1) Method1() { // ExportedType1Method1 inline
	_ = strings.TrimSpace("")
}

// ExportedType2Method1 heading
func (t ExportedType2) Method1() { // ExportedType2Method1 inline
	_ = runtime.GOOS
}

// ExportedType3Method1 heading
func (t ExportedType3) Method1() { // ExportedType3Method1 inline
	_ = build.Default
}

// ExportedFunc1 heading
func ExportedFunc1() { // ExportedFunc1 inline
	_ = strings.TrimSpace("")
}

// ExportedFunc2 heading
func ExportedFunc2() { // ExportedFunc2 inline
	_ = runtime.GOOS
}

// ExportedFunc3 heading
func ExportedFunc3() { // ExportedFunc3 inline
	_ = build.Default
}
