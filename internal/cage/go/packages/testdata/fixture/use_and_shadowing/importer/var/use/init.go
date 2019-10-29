package use

import other_pkg "fixture.tld/use_and_shadowing/pkglocal/var/use"

func init() {
	var a string = other_pkg.ExportedVar1
	_ = a
}
