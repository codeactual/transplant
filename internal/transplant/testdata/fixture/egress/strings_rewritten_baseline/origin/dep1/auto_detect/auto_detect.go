package auto_detect

import "origin.tld/user/proj/dep1/go_descendant"

const RewrittenDepImportPath = "origin.tld/user/proj/dep1/auto_detect"

// rewritten dep path: origin.tld/user/proj/dep1/auto_detect
func AutoDetectFunc1() {
	go_descendant.GoDescendantFunc1()
}
