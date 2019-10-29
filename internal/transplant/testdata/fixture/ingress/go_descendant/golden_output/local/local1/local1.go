package local1

import "origin.tld/user/proj/local/candidates/match"

const (
	RewrittenLocalImportPath = "origin.tld/user/proj/local/local1"
)

func ExportedFunc1() {
	match.MatchFunc1()
	_ = "(edit)"
}
