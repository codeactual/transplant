package dep1

func ExportedFunc1() {
}

func ExportedFunc2() {
}

func ExportedFunc3() { // should get pruned
}

func ExportedFunc4() {
}
