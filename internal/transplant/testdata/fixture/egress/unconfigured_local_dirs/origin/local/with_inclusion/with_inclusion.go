package with_inclusion

import (
	"origin.tld/user/proj/local/with_inclusion/auto_detect_exclusion"
	"origin.tld/user/proj/local/without_inclusion"
)

func with_inclusionFunc() {
	auto_detect_exclusion.AutoDetectExclusionFunc()
	without_inclusion.WithoutExclusionFunc()
}
