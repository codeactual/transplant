package main

func init() {
	var ExportedConst1 = ""
	_ = ExportedConst1
}

func WithConst() {
	const nonExportedConst1 string = ""
	const ExportedConst1 string = ""
	_, _ = nonExportedConst1, ExportedConst1

	const (
		nonExportedConst2 = ""
		ExportedConst2    = ""
	)
	_, _ = nonExportedConst2, ExportedConst2
}

func WithVar() {
	var nonExportedConst1 string = ""
	var ExportedConst1 string = ""
	_, _ = nonExportedConst1, ExportedConst1

	nonExportedConst2, ExportedConst2 := "", ""
	_, _ = nonExportedConst2, ExportedConst2

	var (
		ExportedConst3 string
	)
	_ = ExportedConst3
}

func WithType() {
	type ExportedConst1 struct{}
}
