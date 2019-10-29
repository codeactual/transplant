package dep1

type ExportedType1 struct{}

func (t ExportedType1) Method1() {}

type ExportedType2 struct{}

func (t ExportedType2) Method1() string { return "" }

func ExportedFunc1() {}

func ExportedFunc2() string { return "" }
