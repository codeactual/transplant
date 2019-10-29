package dep1

import "copy.tld/user/proj/internal/dep2"

type UsedIface1 interface { // retained because it's used in UsedFunc1
	Method1()
}

type UsedImpl1 struct { // retained because it's used in UsedFunc1
}

func (t UsedImpl1) Method1() {
}

func UsedFunc1() { // retained because it's used in local.ExportedFunc1
	var i UsedIface1
	i = &UsedImpl1{}
	_ = i
}

// local types
var _ UsedIface1 = (*UsedImpl1)(nil) // retained because LHS/RHS are used in UsedFunc1

// imported-type version of the above declarations
var _ dep2.UsedIface1 = (*dep2.UsedImpl1)(nil) // retained because LHS/RHS are used in dep2.UsedFunc1

// multi-declaration version of the above individual declarations
var (
	// local types
	_ UsedIface1 = (*UsedImpl1)(nil) // retained because LHS/RHS are used in UsedFunc1

	// imported-type version of the above declarations
	_ dep2.UsedIface1 = (*dep2.UsedImpl1)(nil) // retained because LHS/RHS are used in dep2.UsedFunc1
)
