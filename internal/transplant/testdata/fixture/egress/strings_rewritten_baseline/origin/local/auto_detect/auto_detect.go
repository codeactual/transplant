package auto_detect

import (
	dep1_auto_detect "origin.tld/user/proj/dep1/auto_detect"
	"origin.tld/user/proj/local/go_descendant"
)

const RewrittenLocalImportPath = "origin.tld/user/proj/local/auto_detect"

// rewritten local path: origin.tld/user/proj/local/auto_detect
// rewritten dep path: origin.tld/user/proj/dep1/auto_detect
func AutoDetectFunc1() {
	go_descendant.GoDescendantFunc1()
	dep1_auto_detect.AutoDetectFunc1()
	_ = dep1_auto_detect.RewrittenDepImportPath
}
