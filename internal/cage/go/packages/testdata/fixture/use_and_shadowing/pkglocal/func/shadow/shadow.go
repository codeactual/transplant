package shadow

func nonExportedFunc1() string { return "" }
func nonExportedFunc2() string { return "" }
func ExportedFunc1() string    { return "" }
func ExportedFunc2() string    { return "" }
func ExportedFunc3() string    { return "" }

func init() {
	var ExportedFunc1 = ""
	_ = ExportedFunc1
}

func WithConst() {
	const nonExportedFunc1 string = ""
	const ExportedFunc1 string = ""
	_, _ = nonExportedFunc1, ExportedFunc1

	const (
		nonExportedFunc2 = ""
		ExportedFunc2    = ""
	)
	_, _ = nonExportedFunc2, ExportedFunc2
}

func WithVar() {
	var nonExportedFunc1 string = ""
	var ExportedFunc1 string = ""
	_, _ = nonExportedFunc1, ExportedFunc1

	nonExportedFunc2, ExportedFunc2 := "", ""
	_, _ = nonExportedFunc2, ExportedFunc2

	var (
		ExportedFunc3 string
	)
	_ = ExportedFunc3
}

func WithType() {
	type ExportedFunc1 struct{}
}

func InParamName(ExportedFunc1 string) {
	ExportedFunc2 := 2
	_ = ExportedFunc2
}
