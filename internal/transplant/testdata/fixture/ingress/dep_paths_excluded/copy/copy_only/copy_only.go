package copy_only

import "trigger_error_if_both_copied_and_inspected"

const (
	RewrittenLocalImportPath = "copy.tld/user/proj"
	RewrittenDepImportPath   = "copy.tld/user/proj/internal/dep1"
)

func CopyOnlyNewFunc() {
}
