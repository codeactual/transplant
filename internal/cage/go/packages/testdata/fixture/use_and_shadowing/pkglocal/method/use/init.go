package use

func init() {
	a := nonExportedType1{}.Method1
	b := ExportedType1{}.Method1()
	_, _ = a, b

	c := ExportedType3{}
	c.Method1()
}
