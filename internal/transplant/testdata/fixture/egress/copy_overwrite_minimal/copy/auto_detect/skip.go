package auto_detect

import (
	local_go_descendant "copy.tld/user/proj/go_descendant"
	"copy.tld/user/proj/internal/dep1/auto_detect"
	dep_go_descendant "copy.tld/user/proj/internal/dep1/go_descendant"
)

func SkipFunc() {
	auto_detect.SkipFunc()
	auto_detect.ChangedFunc()
	local_go_descendant.GoDescendantFunc()
	dep_go_descendant.GoDescendantFunc()
}
