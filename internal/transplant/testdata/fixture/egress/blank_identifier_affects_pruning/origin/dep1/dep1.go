package dep1

import "origin.tld/user/proj/dep2"

type UsedIface1 interface { // retained because it's used in UsedFunc1
	Method1()
}

type UnusedIface1 interface { // pruned because it's only used in "_" assertion of retained UsedImpl1
	Method1()
}

type UsedImpl1 struct { // retained because it's used in UsedFunc1
}

func (t UsedImpl1) Method1() {
}

type UnusedImpl1 struct { // pruned because it's not used in UsedFunc1
}

func (t UnusedImpl1) Method1() {
}

func UsedFunc1() { // retained because it's used in local.ExportedFunc1
	var i UsedIface1
	i = &UsedImpl1{}
	_ = i
}

// local types
var _ UsedIface1 = (*UsedImpl1)(nil)     // retained because LHS/RHS are used in UsedFunc1
var _ UnusedIface1 = (*UsedImpl1)(nil)   // pruned because it's only used in this line
var _ UsedIface1 = (*UnusedImpl1)(nil)   // pruned because RHS is not used
var _ UnusedIface1 = (*UnusedImpl1)(nil) // pruned because RHS is not used

// imported-type version of the above declarations
var _ dep2.UsedIface1 = (*dep2.UsedImpl1)(nil)     // retained because LHS/RHS are used in dep2.UsedFunc1
var _ dep2.UnusedIface1 = (*dep2.UsedImpl1)(nil)   // pruned because it's only used in this line
var _ dep2.UsedIface1 = (*dep2.UnusedImpl1)(nil)   // pruned because RHS is not used
var _ dep2.UnusedIface1 = (*dep2.UnusedImpl1)(nil) // pruned because RHS is not used

// multi-declaration version of the above individual declarations
var (
	// local types
	_ UsedIface1   = (*UsedImpl1)(nil)   // retained because LHS/RHS are used in UsedFunc1
	_ UnusedIface1 = (*UsedImpl1)(nil)   // pruned because it's only used in this line
	_ UsedIface1   = (*UnusedImpl1)(nil) // pruned because RHS is not used
	_ UnusedIface1 = (*UnusedImpl1)(nil) // pruned because RHS is not used

	// imported-type version of the above declarations
	_ dep2.UsedIface1   = (*dep2.UsedImpl1)(nil)   // retained because LHS/RHS are used in dep2.UsedFunc1
	_ dep2.UnusedIface1 = (*dep2.UsedImpl1)(nil)   // pruned because it's only used in this line
	_ dep2.UsedIface1   = (*dep2.UnusedImpl1)(nil) // pruned because RHS is not used
	_ dep2.UnusedIface1 = (*dep2.UnusedImpl1)(nil) // pruned because RHS is not used
)
