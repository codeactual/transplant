package auto_detect

import (
	"copy.tld/user/proj/go_descendant"
	dep1_auto_detect "copy.tld/user/proj/internal/dep1/auto_detect"
)

const RewrittenLocalImportPath = "copy.tld/user/proj/auto_detect"

// rewritten local path: copy.tld/user/proj/auto_detect
// rewritten dep path: copy.tld/user/proj/internal/dep1/auto_detect
func AutoDetectFunc1() {
	go_descendant.GoDescendantFunc1()
	dep1_auto_detect.AutoDetectFunc1()
	_ = dep1_auto_detect.RewrittenDepImportPath
}
