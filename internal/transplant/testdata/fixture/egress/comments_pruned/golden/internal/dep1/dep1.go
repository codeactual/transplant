package dep1

// Import heading
import "strings"

// Import list heading
import (
	// Import list heading
	"os"

	"sync" // Import list inline
)

type MyInt int // Type inline

type MyStruct struct {
	// Type heading, inner
	S string

	I int // Type inline, innner
}

// Method heading
func (s MyStruct) Method1() {
}

func (s MyStruct) Method2() { // Method inline
	// Not used but should remain because Method1 is used and method sets are retained
	// as a group to avoid interface implementation tracking.
}

// Const heading
const S = ""

// Floating structural -----------------------------------------------------------------------------
// Floating structural

// Func heading
func ExportedFunc1() {
	// Call heading
	ExportedFunc2()

	_ = strings.TrimSpace("") // Call inline

	// Floating structural, inner -----------------------------------------------------------------------------
	// Floating structural, inner

	/**
	 * Var multi-line heading
	 */
	var wg sync.WaitGroup
	_ = wg

	/**
	 * Floating structural, multi-line -----------------------------------------------------------------------------
	 * Floating structural, multi-line
	 */

	_ = os.Stdout // Assignment inline
}

/**
 * Floating structural, multi-line -----------------------------------------------------------------------------
 * Floating structural, multi-line
 */

// Func heading
func ExportedFunc2() { // Func inline
}
