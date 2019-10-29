package use

import other_pkg "fixture.tld/use_and_shadowing/pkglocal/type/use"

func init() {
	var a other_pkg.ExportedType1
	_ = a
}
