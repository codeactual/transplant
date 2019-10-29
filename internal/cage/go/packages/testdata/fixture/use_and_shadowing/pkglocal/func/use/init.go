package use

func init() {
	a := nonExportedFunc1
	b := ExportedFunc1()
	_, _ = a, b
}
