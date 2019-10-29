package dep2

func ExportedFunc1() {
}

func ExportedFunc2() { // should get pruned
}
