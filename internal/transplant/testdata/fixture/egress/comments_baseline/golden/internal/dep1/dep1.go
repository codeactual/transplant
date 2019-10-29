package dep1

// Import heading
import "strings"

import "runtime" // Import inline

// Import list heading
import (
	// Import list heading
	"os"

	"sync" // Import list inline
)

type MyInt int // Type inline

// Type heading
type MyString string

type MyStruct struct {
	// Type heading, inner
	S string

	I int // Type inline, innner
}

// Method heading
func (s MyStruct) Method1() {
}

func (s MyStruct) Method2() { // Method inline
}

const I = 1 // Const inline

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

	// To retain these after pruning
	_, _, _, _, _ = I, S, MyInt(1), MyStruct{}, MyString("")
}

/**
 * Floating structural, multi-line -----------------------------------------------------------------------------
 * Floating structural, multi-line
 */

func ExportedFunc2() { // Func inline
	_ = runtime.GOOS
}
