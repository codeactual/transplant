package main

import const_pkg "fixture.tld/use_and_shadowing/pkglocal/const/use"

func init() {
	_ = ExportedConst2
	_ = const_pkg.ExportedConst3
}
