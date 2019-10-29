package dep1

func Dep1UsedFunc1() {
	Dep1UsedFunc2()
}

func Dep1UsedFunc2(Dep1UnusedFunc1 ...string) {
	// parameter name does not protect the identically named global from pruning

	Dep1Type{}.UsedMethod1()
}

type Dep1Type struct{}

func (Dep1UnusedFunc2 Dep1Type) UsedMethod1(Dep1UnusedFunc3 ...string) {
	// parameter/receiver names do not protect the identically named globals from pruning
}
