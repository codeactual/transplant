package shadow

const (
	nonExportedVar1 = ""
	ExportedVar1    = ""

	nonExportedVar2 = ""
	ExportedVar2    = ""

	ExportedVar3 = ""
)

func init() {
	var ExportedVar1 = ""
	_ = ExportedVar1
}

func WithConst() {
	const nonExportedVar1 string = ""
	const ExportedVar1 string = ""
	_, _ = nonExportedVar1, ExportedVar1

	const (
		nonExportedVar2 = ""
		ExportedVar2    = ""
	)
	_, _ = nonExportedVar2, ExportedVar2
}

func WithVar() {
	var nonExportedVar1 string
	var ExportedVar1 string
	_, _ = nonExportedVar1, ExportedVar1

	nonExportedVar2, ExportedVar2 := "", ""
	_, _ = nonExportedVar2, ExportedVar2

	var (
		ExportedVar3 string
	)
	_ = ExportedVar3
}

func WithType() {
	type ExportedVar1 struct{}
}
