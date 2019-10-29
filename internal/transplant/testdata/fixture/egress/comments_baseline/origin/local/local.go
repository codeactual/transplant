// Copyright line
// Copyright line

package local

import "origin.tld/user/proj/dep1" // Import inline

import "runtime" // Import inline

// Import heading
import "strings"

// Import list heading
import (
	"sync" // Import list inline

	// Import list heading
	"os"
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
	dep1.ExportedFunc1()

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

func ExportedFunc2() { // Func inline
	_ = runtime.GOOS
}
