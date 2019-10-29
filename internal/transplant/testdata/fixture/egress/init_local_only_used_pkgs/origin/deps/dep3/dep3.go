package dep3

var num int

func init() {
	// Force inspection of the function's globals and the code path that would register this file to be copied
	// if it qualifies.
	num = 1
}

func ExportedFunc1() {
}
