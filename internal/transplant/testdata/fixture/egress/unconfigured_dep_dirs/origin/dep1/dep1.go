package dep1

import (
	"origin.tld/user/proj/dep1/auto_detect_exclusion"
	"origin.tld/user/proj/dep_without_inclusion1"
)

func Dep1Func() {
	auto_detect_exclusion.AutoDetectExclusionFunc()
	dep_without_inclusion1.DepWithoutExclusion1Func()
}
