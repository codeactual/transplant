package auto_detect

import (
	"origin.tld/user/proj/dep1/auto_detect"
	dep_go_descendant "origin.tld/user/proj/dep1/go_descendant"
	local_go_descendant "origin.tld/user/proj/local/go_descendant"
)

func SkipFunc() {
	auto_detect.SkipFunc()
	auto_detect.ChangedFunc()
	local_go_descendant.GoDescendantFunc()
	dep_go_descendant.GoDescendantFunc()
}
