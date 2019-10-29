package shadow

type nonExportedType1 struct{}
type nonExportedType2 struct{}
type ExportedType1 struct{}
type ExportedType2 struct{}
type ExportedType3 struct{}

func init() {
	var ExportedType1 = ""
	_ = ExportedType1
}

func WithConst() {
	const nonExportedType1 string = ""
	const ExportedType1 string = ""
	_, _ = nonExportedType1, ExportedType1

	const (
		nonExportedType2 = ""
		ExportedType2    = ""
	)
	_ = ExportedType2
}

func WithVar() {
	var nonExportedType1 string = ""
	var ExportedType1 string = ""
	_, _ = nonExportedType1, ExportedType1

	nonExportedType2, ExportedType2 := "", ""
	_, _ = nonExportedType2, ExportedType2

	var (
		ExportedType3 string
	)
	_ = ExportedType3
}

func WithType() {
	type ExportedType1 struct{}
}
