package auto_detect

import "copy.tld/user/proj/internal/dep1/go_descendant"

const RewrittenDepImportPath = "copy.tld/user/proj/internal/dep1/auto_detect"

// rewritten dep path: copy.tld/user/proj/internal/dep1/auto_detect
func AutoDetectFunc1() {
	go_descendant.GoDescendantFunc1()
}
