package local1

import "copy.tld/user/proj/candidates/match"

const (
	RewrittenLocalImportPath = "copy.tld/user/proj/local1"
)

func ExportedFunc1() {
	match.MatchFunc1()
	_ = "(edit)"
}
