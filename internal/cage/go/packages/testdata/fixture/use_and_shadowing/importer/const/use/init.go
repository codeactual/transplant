package use

import other_pkg "fixture.tld/use_and_shadowing/pkglocal/const/use"

func init() {
	var a string = other_pkg.ExportedConst1
	_ = a
}
