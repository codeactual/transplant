package use

import other_pkg "fixture.tld/use_and_shadowing/pkglocal/method/use"

func init() {
	a := other_pkg.ExportedType2{}.Method1
	b := other_pkg.ExportedType1{}.Method1()
	_, _ = a, b

	c := other_pkg.ExportedType3{}
	c.Method1()
}
