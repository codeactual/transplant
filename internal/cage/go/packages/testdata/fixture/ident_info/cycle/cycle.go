// package cycle defines types whose dependencies include cycles that must be handled
// without causing stack overflows in the recursion-heavy logic.
//
// The three contrived types are maximally cyclical/interdependent in order to exercise
// the most cases.
//
// Pointers are used to avoid "invalid recursive type" errors.
package cycle

// type (
// nonExportedDefinedInt        int
// anotherNonExportedDefinedInt nonExportedDefinedInt
// )

type A struct {
	*A
	*B
	*C
}

type B struct {
	*A
	*B
	*C
}

type C struct {
	*A
	*B
	*C
}
