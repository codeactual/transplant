package cpy_only

import "trigger_error_if_both_copied_and_inspected"

const (
	RewrittenLocalImportPath = "origin.tld/user/proj"
	RewrittenDepImportPath   = "origin.tld/user/proj/dep1"
)

func CopyOnlyNewFunc() {
}
