package dep2

type UsedIface1 interface { // retained because it's used in UsedFunc1
	Method1()
}

type UnusedIface1 interface { // pruned because it's only used in this line
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
