package use

import other_pkg "fixture.tld/use_and_shadowing/pkglocal/func/use"

func init() {
	a := other_pkg.ExportedFunc2
	b := other_pkg.ExportedFunc1()
	_, _ = a, b
}
